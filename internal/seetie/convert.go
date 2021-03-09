package seetie

import (
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

var validKey = regexp.MustCompile("[a-zA-Z_][a-zA-Z0-9_-]*")

func ExpToMap(ctx *hcl.EvalContext, exp hcl.Expression) (map[string]interface{}, hcl.Diagnostics) {
	val, diags := exp.Value(ctx)
	if SetSeverityLevel(diags).HasErrors() {
		return nil, filterErrors(diags)
	}
	return ValueToMap(val), nil
}

func ValueToMap(val cty.Value) map[string]interface{} {
	result := make(map[string]interface{})
	if val.IsNull() || !val.IsKnown() {
		return result
	}
	var valMap map[string]cty.Value
	if isTuple(val) {
		valMap = val.AsValueSlice()[0].AsValueMap()
	} else {
		valMap = val.AsValueMap()
	}

	for k, v := range valMap {
		if v.IsNull() || !v.IsWhollyKnown() {
			result[k] = ""
			continue
		}
		switch v.Type() {
		case cty.Bool:
			result[k] = v.True()
		case cty.String:
			result[k] = v.AsString()
		case cty.List(cty.String):
			result[k] = ValueToStringSlice(v)
		case cty.Number:
			f, _ := v.AsBigFloat().Float64()
			result[k] = f
		case cty.Map(cty.NilType):
			result[k] = ""
		default:
			if isTuple(v) {
				result[k] = ValueToStringSlice(v)
				continue
			}
			// unknown types results in empty string which gets removed later on
			result[k] = ""
		}
	}
	return result
}

func ValuesMapToValue(m url.Values) cty.Value {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return MapToValue(result)
}

func ListToValue(l []interface{}) cty.Value {
	if len(l) == 0 {
		return cty.NilVal
	}
	var list []cty.Value
	for _, v := range l {
		list = append(list, GoToValue(v))
	}
	return cty.TupleVal(list)
}

func GoToValue(v interface{}) cty.Value {
	switch v.(type) {
	case string:
		return cty.StringVal(ToString(v))
	case bool:
		return cty.BoolVal(v.(bool))
	case float64:
		return cty.NumberFloatVal(v.(float64))
	case []interface{}:
		return ListToValue(v.([]interface{}))
	case map[string]interface{}:
		return MapToValue(v.(map[string]interface{}))
	}
	return cty.NullVal(cty.String)
}

func MapToValue(m map[string]interface{}) cty.Value {
	if m == nil {
		return cty.MapValEmpty(cty.NilType)
	}

	ctyMap := make(map[string]cty.Value)

	for k, v := range m {
		if !validKey.MatchString(k) {
			continue
		}
		switch v.(type) {
		case []string:
			var list []interface{}
			for _, s := range v.([]string) {
				list = append(list, s)
			}
			ctyMap[k] = ListToValue(list)
		case []interface{}:
			ctyMap[k] = ListToValue(v.([]interface{}))
		case map[string]interface{}:
			ctyMap[k] = MapToValue(v.(map[string]interface{}))
		default:
			ctyMap[k] = GoToValue(v)
		}
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.NilType) // prevent attribute access on nil values
	}

	return cty.ObjectVal(ctyMap)
}

func HeaderToMapValue(headers http.Header) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range headers {
		if validKey.MatchString(k) {
			if len(v) == 0 {
				ctyMap[strings.ToLower(k)] = cty.StringVal("")
				continue
			}
			ctyMap[strings.ToLower(k)] = cty.StringVal(v[0]) // TODO: ListVal??
		}
	}
	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func CookiesToMapValue(cookies []*http.Cookie) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, cookie := range cookies {
		ctyMap[cookie.Name] = cty.StringVal(cookie.Value) // TODO: ListVal??
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

func ValueToStringSlice(src cty.Value) []string {
	var l []string
	switch src.Type() {
	case cty.NilType:
		return l
	case cty.Bool, cty.Number, cty.String:
		return append(l, ValueToString(src))
	default:
		for _, s := range src.AsValueSlice() {
			if !s.IsKnown() {
				continue
			}
			l = append(l, ValueToString(s))
		}
	}
	return l
}

var whitespaceRegex = regexp.MustCompile(`^\s*$`)

// ValueToString explicitly drops all other (unknown) types and
// converts non whitespace strings or numbers to its string representation.
func ValueToString(v cty.Value) string {
	if v.IsNull() || !v.IsKnown() {
		return ""
	}

	switch v.Type() {
	case cty.String:
		str := v.AsString()
		if whitespaceRegex.MatchString(str) {
			return ""
		}
		return str
	case cty.Number:
		n := v.AsBigFloat()
		ni, accuracy := n.Int(nil)
		if accuracy == big.Exact {
			return ni.String()
		}
		return n.String()
	default:
		return ""
	}
}

func ValueToInt(v cty.Value) int64 {
	n := v.AsBigFloat()
	ni, _ := n.Int64()
	return ni
}

func SliceToString(sl []interface{}) string {
	var str []string
	for _, s := range sl {
		if result := ToString(s); result != "" {
			str = append(str, result)
		}
	}
	return strings.Join(str, ",")
}

func ToString(s interface{}) string {
	switch s.(type) {
	case []string:
		return strings.Join(s.([]string), ",")
	case []interface{}:
		return SliceToString(s.([]interface{}))
	case string:
		return s.(string)
	case int:
		return strconv.Itoa(s.(int))
	case float64:
		return fmt.Sprintf("%0.f", s)
	case bool:
		if !s.(bool) {
			return "false"
		}
		return "true"
	default:
	}
	return ""
}

// isTuple checks by type name since tuple is not comparable by type.
func isTuple(v cty.Value) bool {
	if v.IsNull() {
		return false
	}
	return v.Type().FriendlyNameForConstraint() == "tuple"
}

func SetSeverityLevel(diags hcl.Diagnostics) hcl.Diagnostics {
	for _, d := range diags {
		switch d.Summary {
		case "Missing map element", "Unsupported attribute":
			d.Severity = hcl.DiagWarning
		}
	}
	return diags
}

func filterErrors(diags hcl.Diagnostics) hcl.Diagnostics {
	var errs hcl.Diagnostics
	for _, err := range diags.Errs() {
		errs = append(errs, err.(*hcl.Diagnostic))
	}
	return errs
}
