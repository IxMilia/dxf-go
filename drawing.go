package dxf

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

// The Drawing struct represents a complete DXF drawing.
type Drawing struct {
	Header Header

	AppIds       []AppId
	BlockRecords []BlockRecord
	DimStyles    []DimStyle
	Layers       []Layer
	LineTypes    []LineType
	Styles       []Style

	Entities []Entity
}

// NewDrawing returns a new, fully initialized drawing.
func NewDrawing() *Drawing {
	return &Drawing{
		Header:   *NewHeader(),
		Entities: make([]Entity, 0),
	}
}

// SaveFile writes the current drawing to the specified path.
func (d *Drawing) SaveFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return d.SaveToWriter(f)
}

// SaveToWriter writes the current drawing to the specified io.Writer.
func (d *Drawing) SaveToWriter(writer io.Writer) error {
	codePairWriter := newASCIICodePairWriter(writer)
	return d.saveToCodePairWriter(codePairWriter)
}

func (d *Drawing) String() string {
	buf := new(bytes.Buffer)
	err := d.SaveToWriter(buf)
	if err != nil {
		return err.Error()
	}

	return buf.String()
}

func (d *Drawing) saveToCodePairWriter(writer codePairWriter) error {
	err := d.Header.writeHeaderSection(writer)
	if err != nil {
		return err
	}

	err = writeTablesSection(d, writer, d.Header.Version)
	if err != nil {
		return err
	}

	err = writeEntitiesSection(d.Entities, writer, d.Header.Version)
	if err != nil {
		return err
	}

	err = writer.writeCodePair(NewStringCodePair(0, "EOF"))
	return err
}

// ReadFile reads a DXF drawing from the specified path.
func ReadFile(path string) (Drawing, error) {
	var drawing Drawing
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return drawing, err
	}

	return ReadFromReader(bytes.NewReader(buf))
}

// ReadFromReader reads a DXF drawing from the specified io.Reader.
func ReadFromReader(reader io.Reader) (Drawing, error) {
	codePairReader := newASCIICodePairReader(reader)
	return readFromCodePairReader(codePairReader)
}

// ParseDrawing returns a drawing as parsed from a `string`.
func ParseDrawing(content string) (Drawing, error) {
	stringReader := strings.NewReader(content)
	return ReadFromReader(stringReader)
}

func readFromCodePairReader(reader codePairReader) (Drawing, error) {
	drawing := *NewDrawing()

	// read sections
	nextPair, err := reader.readCodePair()

	// parse sections
	for err == nil && !nextPair.isEOF() {
		if !nextPair.isStartSection() {
			return drawing, errors.New("expected 0/SECTION code pair")
		}

		// find 2/<section-type>
		nextPair, err = reader.readCodePair()
		if err != nil {
			return drawing, err
		}
		if nextPair.Code != 2 {
			return drawing, errors.New("expected 2/<section-type>")
		}

		sectionType := nextPair.Value.(StringCodePairValue).Value
		nextPair, err = reader.readCodePair()
		for err == nil && !nextPair.isEndSection() {
			switch sectionType {
			case "ENTITIES":
				drawing.Entities, nextPair, err = readEntities(nextPair, reader)
			case "HEADER":
				drawing.Header, nextPair, err = readHeader(nextPair, reader)
			case "TABLES":
				nextPair, err = readTables(&drawing, nextPair, reader)
			default:
				// swallow unsupported section
				for err == nil && !nextPair.isEndSection() {
					nextPair, err = reader.readCodePair()
				}
			}
		}

		// find 0/ENDSEC
		if err != nil {
			return drawing, err
		}
		if !nextPair.isEndSection() {
			return drawing, errors.New("expected 0/ENDSEC")
		}

		nextPair, err = reader.readCodePair()
	}

	// find possible 0/EOF
	if err != nil {
		// don't care at this point, the file could be done
		return drawing, nil
	}
	if !nextPair.isEOF() {
		return drawing, errors.New("expected 0/EOF")
	}

	return drawing, nil
}
