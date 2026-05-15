package parser

import (
	"archive/zip"
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

const hexPrefix = "0x"

type Parser struct {
	log     *slog.Logger
	db      DB
	dataDir string
}

func New(log *slog.Logger, db DB, dataDir string) *Parser {
	return &Parser{
		log:     log,
		db:      db,
		dataDir: dataDir,
	}
}

func (s *Parser) Parse(ctx context.Context, path string) (int64, error) {
	const op = "parser.Parse"

	log := s.log.With(slog.String("op", op), slog.String("path", path))

	if !strings.HasPrefix(path, s.dataDir+"/") {
		log.Error("path traversal detected")
		return 0, fmt.Errorf("%s: %w", op, ErrPathTraversal)
	}

	reader, err := zip.OpenReader(path)
	if err != nil {
		log.Error("failed to open archive", slog.String("error", err.Error()))
		if errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("%s: %w", op, ErrArchiveNotFound)
		}
		return 0, fmt.Errorf("%s: %w", op, ErrInvalidArchive)
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			s.log.Error("failed to close archive", "error", err)
		}
	}()

	var dbCSVFile, sharpInfoFile *zip.File
	for _, f := range reader.File {
		if err = ctx.Err(); err != nil {
			return 0, err
		}

		switch {
		case strings.HasSuffix(f.Name, ".db_csv"):
			dbCSVFile = f
		case strings.HasSuffix(f.Name, ".sharp_an_info"):
			sharpInfoFile = f
		}
	}

	if dbCSVFile == nil || sharpInfoFile == nil {
		return 0, fmt.Errorf("%s: required files not found in archive: %w", op, ErrRequiredFilesNotFound)
	}

	nodes, ports, nodesInfo, err := s.parseDBCSV(ctx, dbCSVFile)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	sharpData, err := s.parseSharpInfo(ctx, sharpInfoFile)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	for i := range nodesInfo {
		if sharp, ok := sharpData[nodesInfo[i].NodeGUID]; ok {
			nodesInfo[i].Endianness = sharp.Endianness
			nodesInfo[i].ReproducibilityDisable = sharp.ReproducibilityDisable
		}
	}

	if err := s.validate(nodes, ports, nodesInfo); err != nil {
		return 0, fmt.Errorf("%s: validation failed: %w", op, err)
	}

	logData := &LogData{
		Filename:  path,
		Nodes:     nodes,
		Ports:     ports,
		NodesInfo: nodesInfo,
	}

	logID, err := s.db.SaveLogData(ctx, logData)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("successfully parsed", slog.Int64("log_id", logID))
	return logID, nil
}

func (s *Parser) parseDBCSV(ctx context.Context, file *zip.File) ([]Node, []Port, []NodeInfo, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open db_csv: %w", err)
	}
	defer func() {
		err = rc.Close()
		if err != nil {
			s.log.Error("failed to close db_csv", "error", err)
		}
	}()

	reader := csv.NewReader(rc)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	var (
		nodes          []Node
		ports          []Port
		nodesInfo      []NodeInfo
		switchInfo     = make(map[string]NodeInfo)
		currentSection string
	)

	for {
		if err = ctx.Err(); err != nil {
			return nil, nil, nil, err
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read csv: %w: %w", ErrInvalidCSVFormat, err)
		}

		switch record[0] {
		case "START_NODES":
			currentSection = "nodes"
			_, err = reader.Read()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to skip nodes header: %w", err)
			}
			continue
		case "START_PORTS":
			currentSection = "ports"
			_, err = reader.Read()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to skip ports header: %w", err)
			}
			continue
		case "START_SWITCHES":
			currentSection = "switches"
			_, err = reader.Read()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to skip switches header: %w", err)
			}
			continue
		case "START_SYSTEM_GENERAL_INFORMATION":
			currentSection = "system_info"
			_, err = reader.Read()
			if err != nil {
				return nil, nil, nil, fmt.Errorf("failed to skip system_info header: %w", err)
			}
			continue
		case "END_NODES", "END_PORTS", "END_SWITCHES", "END_SYSTEM_GENERAL_INFORMATION":
			currentSection = ""
			continue
		}

		switch currentSection {
		case "nodes":
			if len(record) < 8 {
				return nil, nil, nil, fmt.Errorf("%w", ErrInvalidNodeRecord)
			}
			nodeGUID := trimHexPrefix(strings.Trim(record[6], "\""))
			nodes = append(nodes, Node{
				NodeDesc: strings.Trim(record[0], "\""),
				NumPorts: parseInt(record[1]),
				NodeType: parseInt(record[2]),
				NodeGUID: nodeGUID,
			})

		case "ports":
			if len(record) < 21 {
				continue
			}
			nodeGUID := trimSpaceHexPrefix(record[0])
			portGUID := trimSpaceHexPrefix(record[1])
			ports = append(ports, Port{
				NodeGUID:      nodeGUID,
				PortGUID:      portGUID,
				PortNum:       parseInt(record[2]),
				LID:           parseInt(record[6]),
				LinkWidthActv: parseInt(record[10]),
				LinkSpeedActv: parseInt(record[15]),
				PortState:     parseInt(record[20]),
			})

		case "switches":
			if len(record) < 13 {
				continue
			}
			nodeGUID := trimSpaceHexPrefix(record[0])
			switchInfo[nodeGUID] = NodeInfo{
				NodeGUID:     nodeGUID,
				LinearFDBCap: parseInt(record[1]),
				MCastFDBCap:  parseInt(record[3]),
				LidsPerPort:  parseInt(record[11]),
			}

		case "system_info":
			if len(record) < 5 {
				continue
			}
			nodeGUID := trimSpaceHexPrefix(record[0])
			nodesInfo = append(nodesInfo, NodeInfo{
				NodeGUID:     nodeGUID,
				SerialNumber: strings.Trim(record[1], "\""),
				PartNumber:   strings.Trim(record[2], "\""),
				ProductName:  strings.Trim(record[4], "\""),
			})
		}
	}

	for i := range nodesInfo {
		if sw, ok := switchInfo[nodesInfo[i].NodeGUID]; ok {
			nodesInfo[i].LinearFDBCap = sw.LinearFDBCap
			nodesInfo[i].MCastFDBCap = sw.MCastFDBCap
			nodesInfo[i].LidsPerPort = sw.LidsPerPort
		}
	}

	for guid, sw := range switchInfo {
		found := false
		for _, info := range nodesInfo {
			if info.NodeGUID == guid {
				found = true
				break
			}
		}
		if !found {
			nodesInfo = append(nodesInfo, sw)
		}
	}

	return nodes, ports, nodesInfo, nil
}

func (s *Parser) parseSharpInfo(ctx context.Context, file *zip.File) (map[string]NodeInfo, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open sharp_an_info: %w", err)
	}
	defer func() {
		err = rc.Close()
		if err != nil {
			s.log.Error("failed to close sharp_an_info file", "error", err)
		}
	}()

	result := make(map[string]NodeInfo)
	scanner := bufio.NewScanner(rc)

	var currentGUID string
	var currentInfo NodeInfo

	for scanner.Scan() {
		if err = ctx.Err(); err != nil {
			return nil, err
		}

		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "SW_GUID=") {
			if currentGUID != "" {
				result[currentGUID] = currentInfo
			}
			currentGUID = trimHexPrefix(strings.TrimPrefix(line, "SW_GUID="))
			currentGUID = strings.TrimSpace(currentGUID)
			currentInfo = NodeInfo{NodeGUID: currentGUID}
			continue
		}

		if currentGUID == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "endianness":
			currentInfo.Endianness = parseInt(value)
		case "reproducibility_disable":
			currentInfo.ReproducibilityDisable = parseInt(value)
		}
	}

	if currentGUID != "" {
		result[currentGUID] = currentInfo
	}

	return result, scanner.Err()
}

func (s *Parser) validate(nodes []Node, ports []Port, nodesInfo []NodeInfo) error {
	nodeGUIDs := make(map[string]bool)
	for _, node := range nodes {
		nodeGUIDs[trimHexPrefix(node.NodeGUID)] = true
	}

	for _, port := range ports {
		if !nodeGUIDs[trimHexPrefix(port.NodeGUID)] {
			return fmt.Errorf("port references non-existent node: %s", port.NodeGUID)
		}
	}

	for _, info := range nodesInfo {
		if !nodeGUIDs[trimHexPrefix(info.NodeGUID)] {
			return fmt.Errorf("node info references non-existent node: %s", info.NodeGUID)
		}
	}

	return nil
}

func trimSpaceHexPrefix(s string) string {
	return trimHexPrefix(strings.TrimSpace(s))
}

func trimHexPrefix(s string) string {
	return strings.TrimPrefix(s, hexPrefix)
}

func parseInt(s string) int {
	val := 0
	_, err := fmt.Sscanf(s, "%d", &val)
	if err != nil {
		return 0
	}
	return val
}
