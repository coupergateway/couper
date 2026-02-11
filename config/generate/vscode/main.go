//go:build exclude

// Package main generates JSON schema for the VS Code extension.
// Run with: go run ./config/generate/vscode/... -o vscode-schema.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/coupergateway/couper/config/generate/shared"
	"github.com/coupergateway/couper/errors"
)

// Schema is the top-level output structure
type Schema struct {
	Version    string                `json:"version"`
	Blocks     map[string]*Block     `json:"blocks"`
	Attributes map[string]*Attribute `json:"attributes"`
	Functions  map[string]*Function  `json:"functions"`
	Variables  map[string]*Variable  `json:"variables"`
}

// Block represents an HCL block definition
type Block struct {
	Parents       []string `json:"parents,omitempty"`
	Description   string   `json:"description,omitempty"`
	Labels        []string `json:"labels,omitempty"`
	Labelled      *bool    `json:"labelled,omitempty"`
	LabelOptional bool     `json:"labelOptional,omitempty"`
	Docs          string   `json:"docs,omitempty"`
}

// Attribute represents an HCL attribute definition
type Attribute struct {
	Parents        []string    `json:"parents,omitempty"`
	Description    string      `json:"description,omitempty"`
	Type           interface{} `json:"type,omitempty"`
	Options        []string    `json:"options,omitempty"`
	DefiningBlocks []string    `json:"definingBlocks,omitempty"`
	Deprecated     *Deprecated `json:"deprecated,omitempty"`
}

// Deprecated marks an attribute as deprecated
type Deprecated struct {
	Version   string `json:"version,omitempty"`
	Attribute string `json:"attribute,omitempty"`
}

// Function represents an HCL function
type Function struct {
	Description string `json:"description,omitempty"`
}

// Variable represents a context variable
type Variable struct {
	Parents     []string `json:"parents,omitempty"`
	Description string   `json:"description,omitempty"`
	Child       string   `json:"child,omitempty"`
	Values      []string `json:"values,omitempty"`
}

func main() {
	outputFile := flag.String("o", "vscode-schema.json", "Output file path")
	flag.Parse()

	schema := &Schema{
		Version:    "1.0.0",
		Blocks:     make(map[string]*Block),
		Attributes: make(map[string]*Attribute),
		Functions:  make(map[string]*Function),
		Variables:  make(map[string]*Variable),
	}

	// Extract blocks and attributes from config structs
	extractBlocks(schema)
	extractAttributes(schema)

	// Extract functions from eval/context.go
	extractFunctions(schema)

	// Extract variables
	extractVariables(schema)

	// Extract error handler labels
	extractErrorHandlerLabels(schema)

	// Write output
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshaling schema: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Schema written to %s\n", *outputFile)
}

// blockParents tracks parent-child relationships between blocks
var blockParents = make(map[string][]string)

func extractBlocks(schema *Schema) {
	// First pass: collect all blocks and their nested block children
	for _, info := range shared.GetAllConfigStructsForVSCode() {
		blockName := info.BlockName
		schema.Blocks[blockName] = &Block{}

		fields := shared.GetInlineFields(info.Impl)
		for _, field := range fields {
			hclInfo := shared.ParseHCLTag(field.Tag.Get("hcl"))
			if hclInfo.IsBlock && hclInfo.Name != "" {
				childBlockName := normalizeBlockName(hclInfo.Name)
				blockParents[childBlockName] = appendUnique(blockParents[childBlockName], blockName)
			}
		}
	}

	// Second pass: populate block metadata
	for _, info := range shared.GetAllConfigStructsForVSCode() {
		blockName := info.BlockName
		block := schema.Blocks[blockName]

		// Set parents
		if parents, ok := blockParents[blockName]; ok {
			block.Parents = parents
		}

		// Extract description from docs tag on the block field in parent
		block.Description = getBlockDescription(blockName)

		// Check for labels
		fields := shared.GetInlineFields(info.Impl)
		for _, field := range fields {
			hclInfo := shared.ParseHCLTag(field.Tag.Get("hcl"))
			if hclInfo.IsLabel {
				if hclInfo.LabelOptional {
					block.LabelOptional = true
				} else {
					labelled := true
					block.Labelled = &labelled
				}
				break
			}
		}
	}

	// Add special blocks not in config registry
	addSpecialBlocks(schema)
}

func extractAttributes(schema *Schema) {
	bracesRegex := regexp.MustCompile(`{([^}]*)}`)

	for _, info := range shared.GetAllConfigStructsForVSCode() {
		blockName := info.BlockName
		fields := shared.GetInlineFields(info.Impl)

		for _, field := range fields {
			if field.Tag.Get("docs") == "" {
				continue
			}

			hclInfo := shared.ParseHCLTag(field.Tag.Get("hcl"))
			if hclInfo.Name == "" || hclInfo.IsBlock || hclInfo.IsLabel || hclInfo.IsRemain {
				continue
			}

			attrName := hclInfo.Name

			// Get or create attribute
			attr, exists := schema.Attributes[attrName]
			if !exists {
				attr = &Attribute{}
				schema.Attributes[attrName] = attr
			}

			// Add this block as a parent
			attr.Parents = appendUnique(attr.Parents, blockName)

			// Set description (only if not already set)
			if attr.Description == "" {
				desc := field.Tag.Get("docs")
				desc = bracesRegex.ReplaceAllString(desc, "`$1`")
				attr.Description = desc
			}

			// Set type
			if attr.Type == nil {
				attr.Type = getAttributeType(field)
			}

			// Extract options from docs tag
			if options := extractOptions(field.Tag.Get("docs")); len(options) > 0 {
				attr.Options = options
			}

			// Check for defining blocks (label references)
			if isLabelReference(attrName) {
				attr.DefiningBlocks = getDefiningBlocks(attrName)
			}
		}
	}

	// Sort parents for consistent output
	for _, attr := range schema.Attributes {
		sort.Strings(attr.Parents)
	}
}

func extractFunctions(schema *Schema) {
	// Core functions defined in eval/context.go newFunctionsMap()
	functions := map[string]string{
		"base64_decode":    "Decodes Base64 data, as specified in RFC 4648.",
		"base64_encode":    "Encodes Base64 data, as specified in RFC 4648.",
		"can":              "Tries to evaluate the expression given in its first argument.",
		"coalesce":         "Returns the first of the given arguments that is not null.",
		"contains":         "Determines whether a given list contains a given single value as one of its elements.",
		"default":          "Returns the first of the given arguments that is not null.",
		"join":             "Concatenates together the string elements of one or more lists with a given separator.",
		"json_decode":      "Parses the given JSON string and, if it is valid, returns the value it represents.",
		"json_encode":      "Returns a JSON serialization of the given value.",
		"jwt_sign":         "Creates and signs a JSON Web Token (JWT) from information from a referenced jwt_signing_profile block and additional claims provided as a function parameter.",
		"keys":             "Takes a map and returns a sorted list of the map keys.",
		"length":           "Returns the number of elements in the given collection.",
		"lookup":           "Performs a dynamic lookup into a map.",
		"merge":            "Deep-merges two or more of either objects or tuples. `null` arguments are ignored.",
		"oauth2_authorization_url": "Creates an OAuth2 authorization URL from a referenced OAuth2 AC Block or OIDC Block.",
		"oauth2_verifier":  "Creates a cryptographically random key as specified in RFC 7636.",
		"relative_url":     "Returns a relative URL by retaining path, query and fragment components.",
		"saml_sso_url":     "Creates a SAML SingleSignOn URL (including the SAMLRequest parameter) from a referenced saml block.",
		"set_intersection": "Returns a new set containing the elements that exist in all of the given sets.",
		"split":            "Divides a given string by a given separator.",
		"substr":           "Extracts a sequence of characters from another string.",
		"to_lower":         "Converts a given string to lowercase.",
		"to_number":        "Converts its argument to a number value.",
		"to_upper":         "Converts a given string to uppercase.",
		"trim":             "Removes any whitespace characters from the start and end of the given string.",
		"unixtime":         "Retrieves the current UNIX timestamp in seconds.",
		"url_decode":       "URL-decodes a given string according to RFC 3986.",
		"url_encode":       "URL-encodes a given string according to RFC 3986.",
	}

	for name, desc := range functions {
		schema.Functions[name] = &Function{Description: desc}
	}
}

func buildProps(common []string, extra ...string) []string {
	props := make([]string, 0, len(common)+len(extra))
	props = append(props, common...)
	props = append(props, extra...)
	return props
}

func extractVariables(schema *Schema) {
	commonProps := []string{"body", "context", "cookies", "headers", "json_body"}

	requestProps := buildProps(commonProps,
		"form_body", "host", "id", "method", "origin", "path",
		"path_params", "port", "protocol", "query", "url",
	)

	backendRequestProps := buildProps(commonProps,
		"form_body", "host", "id", "method", "origin", "path",
		"port", "protocol", "query", "url",
	)

	responseProps := buildProps(commonProps, "status")

	schema.Variables["request"] = &Variable{
		Values:      requestProps,
		Description: "Client request data including method, path, headers, query, body, and cookies.",
	}

	schema.Variables["backend"] = &Variable{
		Parents:     []string{"backend"},
		Description: "An object with backend attributes.",
		Values:      []string{"health", "beta_tokens", "beta_token"},
	}

	schema.Variables["backends"] = &Variable{
		Child:       "default",
		Description: "An object with all backends and their attributes.",
		Values:      []string{"health", "beta_tokens", "beta_token"},
	}

	schema.Variables["backend_request"] = &Variable{
		Parents:     []string{"backend"},
		Description: "Holds information about the current backend request.",
		Values:      backendRequestProps,
	}

	schema.Variables["backend_requests"] = &Variable{
		Child:       "default",
		Description: "An object with all backend requests and their attributes.",
		Values:      backendRequestProps,
	}

	schema.Variables["backend_response"] = &Variable{
		Parents: []string{"backend"},
		Values:  responseProps,
	}

	schema.Variables["backend_responses"] = &Variable{
		Child:  "default",
		Values: responseProps,
	}

	schema.Variables["couper"] = &Variable{
		Values: []string{"environment", "version"},
	}

	schema.Variables["env"] = &Variable{
		Description: "The value of an environment variable.",
		Values:      []string{},
	}

	schema.Variables["beta_token_response"] = &Variable{
		Parents:     []string{"beta_token_request"},
		Description: "Holds information about the current token response.",
		Values:      responseProps,
	}
}

// ErrorHandlerLabels stores the valid error labels per parent block
var ErrorHandlerLabels = make(map[string][]string)

func extractErrorHandlerLabels(schema *Schema) {
	// Extract error types from errors.Definitions
	for _, def := range errors.Definitions {
		kinds := def.Kinds()
		if len(kinds) == 0 {
			continue
		}

		// The first kind is the most specific
		specificKind := kinds[0]

		// Add to each context that can handle this error
		contexts := def.Contexts
		if len(contexts) == 0 {
			// Default contexts for errors without explicit context
			contexts = []string{"api", "endpoint"}
		}

		for _, ctx := range contexts {
			ErrorHandlerLabels[ctx] = appendUnique(ErrorHandlerLabels[ctx], specificKind)
		}

		// Also add parent kinds
		for _, kind := range kinds[1:] {
			for _, ctx := range contexts {
				ErrorHandlerLabels[ctx] = appendUnique(ErrorHandlerLabels[ctx], kind)
			}
		}
	}

	// Sort labels for consistent output
	for ctx := range ErrorHandlerLabels {
		sort.Strings(ErrorHandlerLabels[ctx])
	}
}

// Helper functions

func normalizeBlockName(name string) string {
	// Map internal names to HCL block names
	nameMap := map[string]string{
		"beta_health":        "beta_health",
		"beta_rate_limit":    "beta_rate_limit",
		"beta_token_request": "beta_token_request",
		"beta_introspection": "beta_introspection",
	}
	if mapped, ok := nameMap[name]; ok {
		return mapped
	}
	return name
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func getBlockDescription(blockName string) string {
	// Check each config struct for a block field with this name
	for _, info := range shared.GetAllConfigStructsForVSCode() {
		fields := shared.GetInlineFields(info.Impl)
		for _, field := range fields {
			hclInfo := shared.ParseHCLTag(field.Tag.Get("hcl"))
			if hclInfo.IsBlock && normalizeBlockName(hclInfo.Name) == blockName {
				if desc := field.Tag.Get("docs"); desc != "" {
					// Clean up description
					bracesRegex := regexp.MustCompile(`{([^}]*)}`)
					desc = bracesRegex.ReplaceAllString(desc, "`$1`")
					// Remove link references for cleaner descriptions
					linkRegex := regexp.MustCompile(`\[([^\]]+)\]\([^\)]+\)`)
					desc = linkRegex.ReplaceAllString(desc, "$1")
					// Extract first sentence
					if idx := strings.Index(desc, "."); idx > 0 && idx < len(desc)-1 {
						desc = desc[:idx+1]
					}
					return strings.TrimSpace(desc)
				}
			}
		}
	}
	return ""
}

func getAttributeType(field reflect.StructField) interface{} {
	// Check type tag first
	if typeTag := field.Tag.Get("type"); typeTag != "" {
		// Handle "string or object" style types
		if strings.Contains(typeTag, " or ") {
			parts := strings.Split(typeTag, " or ")
			var types []string
			for _, p := range parts {
				p = strings.TrimSpace(p)
				p = strings.Trim(p, "()")
				types = append(types, p)
			}
			return types
		}
		return typeTag
	}

	// Infer from Go type
	return shared.GoTypeToSchemaType(field.Type)
}

var validValuesRegex = regexp.MustCompile(`Valid values?:\s*\{([^}]+)\}`)

func extractOptions(docs string) []string {
	matches := validValuesRegex.FindStringSubmatch(docs)
	if len(matches) < 2 {
		return nil
	}

	optionsStr := matches[1]
	var options []string
	for _, opt := range strings.Split(optionsStr, ",") {
		opt = strings.TrimSpace(opt)
		opt = strings.Trim(opt, `"'{}`)
		if opt != "" {
			options = append(options, opt)
		}
	}
	return options
}

func isLabelReference(attrName string) bool {
	labelRefAttrs := map[string]bool{
		"backend":               true,
		"proxy":                 true,
		"access_control":        true,
		"disable_access_control": true,
		"configuration_backend": true,
		"token_backend":         true,
		"jwks_uri_backend":      true,
		"userinfo_backend":      true,
	}
	return labelRefAttrs[attrName]
}

func getDefiningBlocks(attrName string) []string {
	switch attrName {
	case "backend", "configuration_backend", "token_backend", "jwks_uri_backend", "userinfo_backend":
		return []string{"backend"}
	case "proxy":
		return []string{"proxy"}
	case "access_control", "disable_access_control":
		return []string{"basic_auth", "jwt", "oidc", "saml", "beta_oauth2", "beta_rate_limiter"}
	default:
		return nil
	}
}

func addSpecialBlocks(schema *Schema) {
	// Environment block (preprocessed, can appear in most places)
	schema.Blocks["environment"] = &Block{
		Description:   "Refines the configuration based on the current environment.",
		LabelOptional: false,
		Labelled:      boolPtr(true),
	}

	// Defaults block (top-level only)
	if block, ok := schema.Blocks["defaults"]; ok {
		block.Parents = nil // top-level only
	}

	// Settings block (top-level only)
	if block, ok := schema.Blocks["settings"]; ok {
		block.Parents = nil // top-level only
	}

	// Server block (top-level only)
	if block, ok := schema.Blocks["server"]; ok {
		block.Parents = nil // top-level only
		block.LabelOptional = true
	}

	// Definitions block (top-level only)
	if block, ok := schema.Blocks["definitions"]; ok {
		block.Parents = nil // top-level only
	}
}

func boolPtr(b bool) *bool {
	return &b
}
