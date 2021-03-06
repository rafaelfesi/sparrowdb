package db

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/SparrowDb/sparrowdb/errors"
	"github.com/SparrowDb/sparrowdb/slog"
)

const (
	// DefaultDatabaseConfigFile databases definition
	DefaultDatabaseConfigFile = "database.xml"
)

// DatabaseConfig holds general configuration of database
type DatabaseConfig struct {
	filepath  string
	xmlDbList *XMLDatabaseList
}

// SaveDatabase saves DatabaseDescriptor into the XML file
func (cfg *DatabaseConfig) SaveDatabase(database DatabaseDescriptor) {
	cfg.xmlDbList.Databases = append(cfg.xmlDbList.Databases, database)
	cfg.saveXMLFile()
}

// DropDatabase saves without database into the XML file
func (cfg *DatabaseConfig) DropDatabase(dbname string) {
	for i, v := range cfg.xmlDbList.Databases {
		if v.Name == dbname {
			cfg.xmlDbList.Databases = append(cfg.xmlDbList.Databases[:i],
				cfg.xmlDbList.Databases[i+1:]...)
			cfg.saveXMLFile()
			break
		}
	}
}

func (cfg *DatabaseConfig) saveXMLFile() {
	filePath := filepath.Join(cfg.filepath, DefaultDatabaseConfigFile)

	file, err := os.Create(filePath)
	if err != nil {
		slog.Fatalf(err.Error())
	}

	xmlWriter := io.Writer(file)

	enc := xml.NewEncoder(xmlWriter)
	enc.Indent("  ", "    ")
	if err := enc.Encode(cfg.xmlDbList); err != nil {
		slog.Fatalf(err.Error())
	}
}

// LoadDatabases load DatabaseConfigNode from XML file
func (cfg *DatabaseConfig) LoadDatabases() []DatabaseDescriptor {
	filePath := filepath.Join(cfg.filepath, DefaultDatabaseConfigFile)

	xmlFile, err := os.Open(filePath)
	if err != nil {
		slog.Fatalf(errors.ErrFileNotFound.Error(), filePath)
	}

	defer xmlFile.Close()

	data, err := ioutil.ReadAll(xmlFile)
	if err != nil {
		slog.Fatalf(err.Error())
	}

	descriptor := XMLDatabaseList{}
	if err := xml.Unmarshal(data, &descriptor); err != nil {
		slog.Fatalf(errors.ErrParseFile.Error(), filePath)
	}

	// Put the loaded database list into the sparrowdb instance list
	cfg.xmlDbList.Databases = descriptor.Databases

	v := make([]DatabaseDescriptor, 0, len(cfg.xmlDbList.Databases))

	for _, value := range cfg.xmlDbList.Databases {
		v = append(v, value)
	}

	return v
}

// NewDatabaseConfig return configuration from file
func NewDatabaseConfig(filePath string) *DatabaseConfig {
	return &DatabaseConfig{
		filepath:  filePath,
		xmlDbList: &XMLDatabaseList{},
	}
}
