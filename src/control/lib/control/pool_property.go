//
// (C) Copyright 2021 Intel Corporation.
//
// SPDX-License-Identifier: BSD-2-Clause-Patent
//

package control

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"

	"github.com/daos-stack/daos/src/control/drpc"
)

// PoolProperties returns a map of property names to handlers
// for processing property values.
func PoolProperties() PoolPropertyMap {
	return map[string]*PoolPropHandler{
		"reclaim": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertySpaceReclaim,
				Description: "Reclaim strategy",
			},
			values: map[string]uint64{
				"disabled": drpc.PoolSpaceReclaimDisabled,
				"lazy":     drpc.PoolSpaceReclaimLazy,
				"time":     drpc.PoolSpaceReclaimTime,
			},
		},
		"self_heal": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertySelfHealing,
				Description: "Self-healing policy",
				valueStringer: func(v *PoolPropertyValue) string {
					n, err := v.GetNumber()
					if err != nil {
						return "not set"
					}
					switch {
					case n&drpc.PoolSelfHealingAutoExclude > 0:
						return "exclude"
					case n&drpc.PoolSelfHealingAutoRebuild > 0:
						return "rebuild"
					default:
						return "unknown"
					}
				},
			},
			values: map[string]uint64{
				"exclude": drpc.PoolSelfHealingAutoExclude,
				"rebuild": drpc.PoolSelfHealingAutoRebuild,
			},
		},
		"space_rb": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyReservedSpace,
				Description: "Rebuild space ratio",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					rbErr := errors.Errorf("invalid space_rb value %s (valid values: 0-100)", s)
					rsPct, err := strconv.ParseUint(strings.ReplaceAll(s, "%", ""), 10, 64)
					if err != nil {
						return nil, rbErr
					}
					if rsPct < 0 || rsPct > 100 {
						return nil, rbErr
					}
					return &PoolPropertyValue{rsPct}, nil
				},
				valueStringer: func(v *PoolPropertyValue) string {
					n, err := v.GetNumber()
					if err != nil {
						return "not set"
					}
					return fmt.Sprintf("%d%%", n)
				},
				jsonNumeric: true,
			},
		},
		"label": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyLabel,
				Description: "Pool label",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					if !drpc.LabelIsValid(s) {
						return nil, errors.Errorf("invalid label %q", s)
					}
					return &PoolPropertyValue{s}, nil
				},
			},
		},
		"ec_cell_sz": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyECCellSize,
				Description: "EC cell size",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					b, err := humanize.ParseBytes(s)
					if err != nil {
						return nil, errors.Errorf("invalid EC Cell size %q", s)
					}
					return &PoolPropertyValue{b}, nil
				},
				valueStringer: func(v *PoolPropertyValue) string {
					n, err := v.GetNumber()
					if err != nil {
						return "not set"
					}
					return humanize.IBytes(n)
				},
				jsonNumeric: true,
			},
		},
		"scrub": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyScrubMode,
				Description: "Checksum scrubbing mode",
			},
			values: map[string]uint64{
				"off":        drpc.PoolScrubModeOff,
				"pause":      drpc.PoolScrubModePause,
				"fix":        drpc.PoolScrubModeWait,
				"dynamic": drpc.PoolScrubModeContinuous,
			},
		},
		"scrub-freq": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyScrubFreq,
				Description: "Checksum scrubbing frequency",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					var conv uint64 = 1 // Default unit is seconds

					var unitMap = map[string]uint64 {
						"s":  1,
						"m":  60,
						"h":  60 * 60,
						"d":  60 * 60 * 24,
						"w":  60 * 60 * 168,
					}
					for u, seconds := range unitMap {
						if (strings.HasSuffix(s, u)) {
							s = strings.TrimSuffix(s, u)
							conv = seconds
						}
					}

					rbErr := errors.Errorf("invalid Scrubbing Frequency value %s", s)
					v, err := strconv.ParseUint(s, 10, 64)
					if err != nil {
						return nil, rbErr
					}
					v *= conv
					return &PoolPropertyValue{v}, nil
				},

				valueStringer: func(v *PoolPropertyValue) string {
					freq_seconds, err := v.GetNumber()

					if err != nil {
						return "not set"
					}

					var unit_suffix = []string{"w", "d", "h", "m", "s"}
					var unit_conv = []uint64{60 * 60 * 168, 60 * 60 * 24, 60 * 60, 60, 1}
					result := ""
					for i, u := range unit_suffix {
						seconds := unit_conv[i]
						if freq_seconds >= unit_conv[i] {
							if result != "" {
								result += " "
							}
							result = fmt.Sprintf("%s%d%s", result, freq_seconds/seconds, u)
							freq_seconds %= seconds
						}
					}
					if result == "" {
						result = fmt.Sprintf("%ds", freq_seconds)
					}
					return result
				},
				jsonNumeric: true,
			},
		},
		"scrub-pad": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyScrubPad,
				Description: "Checksum scrubbing padding",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					var conv uint64 = 1000 // Default units is seconds, but store as ms

					var unit_suffix = []string{"ms", "s", "m", "h", "d", "w"}
					var unit_conv = []uint64{1, 1 * 1000, 60 * 1000, 60 * 60 * 1000, 60 * 60 * 24 * 1000, 60 * 60 * 168 * 1000}

					for i, u := range unit_suffix {
						if (strings.HasSuffix(s, u)) {
							conv = unit_conv[i]
							s = strings.TrimSuffix(s, u)
						}
					}

					rbErr := errors.Errorf("invalid Scrubbing Padding value %s", s)
					v, err := strconv.ParseUint(s, 10, 64)
					if err != nil {
						return nil, rbErr
					}
					v *= conv
					return &PoolPropertyValue{v}, nil
				},
				valueStringer: func(v *PoolPropertyValue) string {
					freq_ms, err := v.GetNumber()


					if err != nil {
						return "not set"
					}

					var unit_suffix = []string{"w", "d", "h", "m", "s", "ms" }
					var unit_conv = []uint64{60 * 60 * 168 * 1000, 60 * 60 * 24 * 1000, 60 * 60 * 1000, 60 * 1000, 1 * 1000, 1}
					result := ""
					for i, u := range unit_suffix {
						seconds := unit_conv[i]
						if freq_ms >= unit_conv[i] {
							if result != "" {
								result += " "
							}
							result = fmt.Sprintf("%s%d%s", result, freq_ms/seconds, u)
							freq_ms %= seconds
						}
					}

					if result == "" {
						result = "0"
					}
					return result
				},
				jsonNumeric: true,
			},
		},
		"scrub-cred": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyScrubCred,
				Description: "Checksum scrubbing credits",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					rbErr := errors.Errorf("invalid Scrubbing Credits value %s", s)
					rsPct, err := strconv.ParseUint(strings.ReplaceAll(s, "%", ""), 10, 64)
					if err != nil {
						return nil, rbErr
					}

					return &PoolPropertyValue{rsPct}, nil
				},
				valueStringer: func(v *PoolPropertyValue) string {
					n, err := v.GetNumber()
					if err != nil {
						return "not set"
					}
					return fmt.Sprintf("%d", n)
				},
				jsonNumeric: true,
			},
		},
		"scrub-thresh": {
			Property: PoolProperty{
				Number:      drpc.PoolPropertyScrubThresh,
				Description: "Checksum scrubbing threshold",
				valueHandler: func(s string) (*PoolPropertyValue, error) {
					rbErr := errors.Errorf("invalid Scrubbing Threshold value %s", s)
					rsPct, err := strconv.ParseUint(strings.ReplaceAll(s, "%", ""), 10, 64)
					if err != nil {
						return nil, rbErr
					}

					return &PoolPropertyValue{rsPct}, nil
				},
				valueStringer: func(v *PoolPropertyValue) string {
					n, err := v.GetNumber()
					if err != nil {
						return "not set"
					}
					return fmt.Sprintf("%d", n)
				},
				jsonNumeric: true,
			},
		},
	}
}

// GetProperty returns a *PoolProperty for the property name, if valid.
func (m PoolPropertyMap) GetProperty(name string) (*PoolProperty, error) {
	h, found := m[name]
	if !found {
		return nil, errors.Errorf("unknown property %q", name)
	}
	return h.GetProperty(name), nil
}

// PoolPropertyValue encapsulates the logic for storing a string or uint64
// property value.
type PoolPropertyValue struct {
	data interface{}
}

// SetString sets the property value to a string.
func (ppv *PoolPropertyValue) SetString(strVal string) {
	ppv.data = strVal
}

// SetNumber sets the property value to a number.
func (ppv *PoolPropertyValue) SetNumber(numVal uint64) {
	ppv.data = numVal
}

func (ppv *PoolPropertyValue) String() string {
	if ppv == nil || ppv.data == nil {
		return "value not set"
	}

	switch v := ppv.data.(type) {
	case string:
		return v
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return fmt.Sprintf("unknown data type for %+v", ppv.data)
	}
}

// GetNumber returns the numeric value set for the property,
// or an error if the value is not a number.
func (ppv *PoolPropertyValue) GetNumber() (uint64, error) {
	if ppv == nil || ppv.data == nil {
		return 0, errors.New("value not set")
	}
	if v, ok := ppv.data.(uint64); ok {
		return v, nil
	}
	return 0, errors.Errorf("%+v is not uint64", ppv.data)
}

// PoolProperty contains a name/value pair representing a pool property.
type PoolProperty struct {
	Number        uint32            `json:"-"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Value         PoolPropertyValue `json:"value"`
	jsonNumeric   bool              // true if value should be numeric in JSON
	valueHandler  func(string) (*PoolPropertyValue, error)
	valueStringer func(*PoolPropertyValue) string
}

func (p *PoolProperty) SetValue(strVal string) error {
	if p.valueHandler == nil {
		p.Value.data = strVal
		return nil
	}
	v, err := p.valueHandler(strVal)
	if err != nil {
		return err
	}
	p.Value = *v
	return nil
}

func (p *PoolProperty) String() string {
	if p == nil {
		return "<nil>"
	}

	return p.Name + ":" + p.StringValue()
}

func (p *PoolProperty) StringValue() string {
	if p == nil {
		return "<nil>"
	}
	if p.valueStringer != nil {
		return p.valueStringer(&p.Value)
	}
	return p.Value.String()
}

func (p *PoolProperty) MarshalJSON() ([]byte, error) {
	if p == nil {
		return nil, errors.New("nil property")
	}

	// In some cases, the raw numeric value is preferred
	// for JSON output. Otherwise, just use the string.
	var jsonValue interface{}
	if p.jsonNumeric {
		n, err := p.Value.GetNumber()
		if err != nil {
			return nil, err
		}
		jsonValue = n
	} else {
		jsonValue = p.StringValue()
	}

	type toJSON PoolProperty
	return json.Marshal(&struct {
		*toJSON
		Value interface{} `json:"value"`
	}{
		Value:  jsonValue,
		toJSON: (*toJSON)(p),
	})
}

type valueMap map[string]uint64

func (m valueMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}

type PoolPropHandler struct {
	Property PoolProperty
	values   valueMap
}

func (pph *PoolPropHandler) Values() []string {
	return pph.values.Keys()
}

func (pph *PoolPropHandler) GetProperty(name string) *PoolProperty {
	if pph.Property.valueHandler == nil {
		pph.Property.valueHandler = func(in string) (*PoolPropertyValue, error) {
			if val, found := pph.values[strings.ToLower(in)]; found {
				return &PoolPropertyValue{val}, nil
			}
			return nil, errors.Errorf("invalid value %q for %s (valid: %s)", in, name, strings.Join(pph.Values(), ","))
		}
	}

	if pph.Property.valueStringer == nil {
		valNameMap := make(map[uint64]string)
		for name, number := range pph.values {
			valNameMap[number] = name
		}

		pph.Property.valueStringer = func(v *PoolPropertyValue) string {
			n, err := v.GetNumber()
			if err == nil {
				if name, found := valNameMap[n]; found {
					return name
				}
			}
			return v.String()
		}
	}

	pph.Property.Name = name
	return &pph.Property
}

type PoolPropertyMap map[string]*PoolPropHandler

func (m PoolPropertyMap) Keys() []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
