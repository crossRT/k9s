// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package model1

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

const ageCol = "AGE"

// ExtractionInfo stores data for a field to extract value from another field
type ExtractionInfo struct {
	IdxInFields int
	CustomName  string
	HeaderName  string
	Key         string
}

// ExtractionInfoBag store ExtractionInfo by using the index of the column
type ExtractionInfoBag map[int]ExtractionInfo

// HeaderColumn represent a table header.
type HeaderColumn struct {
	Name      string
	Align     int
	Decorator DecoratorFunc
	Wide      bool
	MX        bool
	Time      bool
	Capacity  bool
	VS        bool
}

// Clone copies a header.
func (h HeaderColumn) Clone() HeaderColumn {
	return h
}

// ----------------------------------------------------------------------------

// Header represents a table header.
type Header []HeaderColumn

func (h Header) Clear() Header {
	h = h[:0]

	return h
}

// Clone duplicates a header.
func (h Header) Clone() Header {
	he := make(Header, 0, len(h))
	for _, h := range h {
		he = append(he, h.Clone())
	}

	return he
}

// Labelize returns a new Header based on labels.
func (h Header) Labelize(cols []int, labelCol int, rr *RowEvents) Header {
	header := make(Header, 0, len(cols)+1)
	for _, c := range cols {
		header = append(header, h[c])
	}
	cc := rr.ExtractHeaderLabels(labelCol)
	for _, c := range cc {
		header = append(header, HeaderColumn{Name: c})
	}

	return header
}

func (h Header) MapIndices(cols []string, wide bool) ([]int, ExtractionInfoBag) {
	var (
		ii   = make([]int, 0, len(cols))
		eib  = make(ExtractionInfoBag)
		rgx  = regexp.MustCompile(`^(?:([^:]+):\s*)?(.*)\[(.*)\]$`)
	)

	for _, col := range cols {
		idx, ok := h.IndexOf(col, true)
		if !ok {
			log.Warn().Msgf("Column %q not found on resource", col)
		}
		
		ii = append(ii, idx)
		
		if !rgx.MatchString(col) {
			continue
		}

		matches := rgx.FindStringSubmatch(col)
		if len(matches) < 4 {
			log.Error().Msgf("Regex match failed for column: %q", col)
			continue
		}
		
		customName := strings.TrimSpace(matches[1]) // For example, GROUP
		headerName := matches[2]                    // For example, LABELS
		key := matches[3]                           // For example, platform.isolation/nodegroup

		if headerName != "LABELS" {
			log.Warn().Msgf("Custom Column %q is not supported", col)
			continue
		}

		log.Info().Msgf("Custom column %q will be displayed as %q", col, customName)

		idxInFields, _ := h.IndexOf(headerName, true)
		eib[len(ii)-1] = ExtractionInfo{idxInFields, customName, headerName, key}
	}

	return ii, eib
}

func (h Header) Customize(cols []string, wide bool) Header {
	if len(cols) == 0 {
		return h
	}

	cc := make(Header, 0, len(h))
	xx := make(map[int]struct{}, len(h))

	// Get column indices and custom name information
	_, extractionInfoBag := h.MapIndices(cols, wide)

	for i, c := range cols {
		idx, ok := h.IndexOf(c, true)
		if !ok {
			cc = append(cc, HeaderColumn{Name: extractionInfoBag[i].CustomName})
			continue
		}
		xx[idx] = struct{}{}

		col := h[idx].Clone()
		col.Wide = false

		cc = append(cc, col)
	}

	if !wide {
		return cc
	}

	// Add wide-column
	for i, c := range h {
		if _, ok := xx[i]; ok {
			continue
		}
		col := c.Clone()
		col.Wide = true
		cc = append(cc, col)
	}

	return cc
}


// Diff returns true if the header changed.
func (h Header) Diff(header Header) bool {
	if len(h) != len(header) {
		return true
	}
	return !reflect.DeepEqual(h, header)
}

// ColumnNames return header col names
func (h Header) ColumnNames(wide bool) []string {
	if len(h) == 0 {
		return nil
	}
	cc := make([]string, 0, len(h))
	for _, c := range h {
		if !wide && c.Wide {
			continue
		}
		cc = append(cc, c.Name)
	}

	return cc
}

// HasAge returns true if table has an age column.
func (h Header) HasAge() bool {
	_, ok := h.IndexOf(ageCol, true)

	return ok
}

// IsMetricsCol checks if given column index represents metrics.
func (h Header) IsMetricsCol(col int) bool {
	if col < 0 || col >= len(h) {
		return false
	}

	return h[col].MX
}

// IsTimeCol checks if given column index represents a timestamp.
func (h Header) IsTimeCol(col int) bool {
	if col < 0 || col >= len(h) {
		return false
	}

	return h[col].Time
}

// IsCapacityCol checks if given column index represents a capacity.
func (h Header) IsCapacityCol(col int) bool {
	if col < 0 || col >= len(h) {
		return false
	}

	return h[col].Capacity
}

// IndexOf returns the col index or -1 if none.
func (h Header) IndexOf(colName string, includeWide bool) (int, bool) {
	for i, c := range h {
		if c.Wide && !includeWide {
			continue
		}
		if c.Name == colName {
			return i, true
		}
	}
	return -1, false
}

// Dump for debugging.
func (h Header) Dump() {
	log.Debug().Msgf("HEADER")
	for i, c := range h {
		log.Debug().Msgf("%d %q -- %t", i, c.Name, c.Wide)
	}
}
