// File based on "github.com/hashicorp/hcl/v2/merged.go" except diagnostic errors for duplicates
// since we explicitly want them and apply an override.

package body

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// reportDuplicates is the global switch to handle force merges.
var reportDuplicates = false

// MergeBodies is like MergeFiles except it deals directly with bodies, rather
// than with entire files.
func MergeBodies(bodies ...hcl.Body) hcl.Body {
	if len(bodies) == 0 {
		// Swap out for our singleton empty body, to reduce the number of
		// empty slices we have hanging around.
		return EmptyBody()
	}

	// If any of the given bodies are already merged bodies, we'll unpack
	// to flatten to a single MergedBodies, since that's conceptually simpler.
	// This also, as a side-effect, eliminates any empty bodies, since
	// empties are merged bodies with no inner bodies.
	var newLen int
	var flatten bool
	for _, body := range bodies {
		if children, merged := body.(MergedBodies); merged {
			newLen += len(children)
			flatten = true
		} else {
			newLen++
		}
	}

	if !flatten { // not just newLen == len, because we might have MergedBodies with single bodies inside
		return MergedBodies(bodies)
	}

	if newLen == 0 {
		// Don't allocate a new empty when we already have one
		return emptyBody
	}

	newBodies := make([]hcl.Body, 0, newLen)
	for _, body := range bodies {
		if children, merged := body.(MergedBodies); merged {
			newBodies = append(newBodies, children...)
		} else {
			newBodies = append(newBodies, body)
		}
	}
	return MergedBodies(newBodies)
}

var emptyBody = MergedBodies([]hcl.Body{})

// EmptyBody returns a body with no content. This body can be used as a
// placeholder when a body is required but no body content is available.
func EmptyBody() hcl.Body {
	return emptyBody
}

type MergedBodies []hcl.Body

// Content returns the content produced by applying the given schema to all
// of the merged bodies and merging the result.
//
// Although required attributes _are_ supported, they should be used sparingly
// with merged bodies since in this case there is no contextual information
// with which to return good diagnostics. Applications working with merged
// bodies may wish to mark all attributes as optional and then check for
// required attributes afterwards, to produce better diagnostics.
func (mb MergedBodies) Content(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Diagnostics) {
	// the returned body will always be empty in this case, because mergedContent
	// will only ever call Content on the child bodies.
	content, _, diags := mb.mergedContent(schema, false)
	return content, diags
}

func (mb MergedBodies) PartialContent(schema *hcl.BodySchema) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	return mb.mergedContent(schema, true)
}

func (mb MergedBodies) JustAttributes() (hcl.Attributes, hcl.Diagnostics) {
	attrs := make(map[string]*hcl.Attribute)
	var diags hcl.Diagnostics

	for _, body := range mb {
		thisAttrs, thisDiags := body.JustAttributes()

		if len(thisDiags) > 0 {
			diags = append(diags, thisDiags...)
		}

		if thisAttrs != nil {
			diags = append(diags, mergeAttributes(attrs, thisAttrs)...)
		}
	}

	return attrs, diags
}

// MissingItemRange returns the first non-empty range if any.
func (mb MergedBodies) MissingItemRange() hcl.Range {
	if len(mb) == 0 {
		// Nothing useful to return here, so we'll return some garbage.
		return hcl.Range{
			Filename: "<empty>",
		}
	}

	for _, b := range mb {
		if be, ok := b.(*hclsyntax.Body); ok {
			if !be.SrcRange.Empty() {
				return be.SrcRange
			}
		} else {
			if !b.MissingItemRange().Empty() {
				return b.MissingItemRange()
			}
		}
	}

	// arbitrarily use the first body's missing item range
	return mb[0].MissingItemRange()
}

func (mb MergedBodies) mergedContent(schema *hcl.BodySchema, partial bool) (*hcl.BodyContent, hcl.Body, hcl.Diagnostics) {
	// We need to produce a new schema with none of the attributes marked as
	// required, since _any one_ of our bodies can contribute an attribute value.
	// We'll separately check that all required attributes are present at
	// the end.
	mergedSchema := &hcl.BodySchema{
		Blocks: schema.Blocks,
	}
	for _, attrS := range schema.Attributes {
		mergedAttrS := attrS
		mergedAttrS.Required = false
		mergedSchema.Attributes = append(mergedSchema.Attributes, mergedAttrS)
	}

	var mergedLeftovers []hcl.Body
	content := &hcl.BodyContent{
		Attributes: map[string]*hcl.Attribute{},
	}

	var diags hcl.Diagnostics
	for _, body := range mb {
		var thisContent *hcl.BodyContent
		var thisLeftovers hcl.Body
		var thisDiags hcl.Diagnostics

		if partial {
			thisContent, thisLeftovers, thisDiags = body.PartialContent(mergedSchema)
		} else {
			thisContent, thisDiags = body.Content(mergedSchema)
		}

		if thisLeftovers != nil {
			mergedLeftovers = append(mergedLeftovers, thisLeftovers)
		}
		if len(thisDiags) > 0 {
			diags = append(diags, thisDiags...)
		}

		if thisContent.Attributes != nil {
			diags = append(diags, mergeAttributes(content.Attributes, thisContent.Attributes)...)
		}

		if len(thisContent.Blocks) != 0 {
			for _, thisContentBlock := range thisContent.Blocks {
				if len(thisContentBlock.Labels) > 0 {
					refs := content.Blocks.OfType(thisContentBlock.Type)
					var mergeLabeled bool
					if len(refs) > 0 {
						thatLabels := refs[0].Labels[:]
						sort.Strings(thatLabels)
						thisLabels := thisContentBlock.Labels[:]
						sort.Strings(thisLabels)

						if strings.Join(thatLabels, "") == strings.Join(thisLabels, "") {
							mergeLabeled = true
						}
					}

					if !mergeLabeled {
						content.Blocks = append(content.Blocks, thisContentBlock)
						continue
					}
				}
				// assume a block definition without a label could not exist twice. Merge attrs.
				var contentBlockType string
				var idx = 0
				for i, contentBlock := range content.Blocks {
					if contentBlock.Type == thisContentBlock.Type {
						contentBlockType = contentBlock.Type
						idx = i
						break
					}
				}

				if contentBlockType == "" { // nothing found
					content.Blocks = append(content.Blocks, thisContentBlock)
					continue
				}

				contentAttrs, contentAttrsDiags := content.Blocks[idx].Body.JustAttributes()
				if contentAttrsDiags.HasErrors() {
					diags = append(diags, contentAttrsDiags...)
				}
				thisContentAttrs, thisContentAttrsDiags := thisContentBlock.Body.JustAttributes()
				if thisContentAttrsDiags.HasErrors() {
					diags = append(diags, thisContentAttrsDiags...)
				}

				diags = append(diags, mergeAttributes(contentAttrs, thisContentAttrs)...)

				content.Blocks[idx].Body = MergeBodies(
					thisContentBlock.Body, // keep child blocks
					New(&hcl.BodyContent{
						Attributes:       contentAttrs,
						MissingItemRange: content.Blocks[idx].DefRange}),
				)
			}
		}
	}

	// Finally, we check for required attributes.
	for _, attrS := range schema.Attributes {
		if !attrS.Required {
			continue
		}

		itemRange := content.MissingItemRange
		if itemRange.Empty() { // use parent range as fallback
			itemRange = mb.MissingItemRange()
		}

		if content.Attributes[attrS.Name] == nil {
			// We don't have any context here to produce a good diagnostic,
			// which is why we warn in the Content docstring to minimize the
			// use of required attributes on merged bodies.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Subject:  &itemRange,
				Summary:  "Missing required argument",
				Detail: fmt.Sprintf(
					"The argument %q is required, but was not set.",
					attrS.Name,
				),
			})
		}
	}

	leftoverBody := MergeBodies(mergedLeftovers...)
	return content, leftoverBody, diags
}

// JustAllAttributes returns a list of attributes in order. Since these bodies got added as partialContent
// we do not have to supply a scheme or handle the underlying diagnostics.
func (mb MergedBodies) JustAllAttributes() []hcl.Attributes {
	return mb.JustAllAttributesWithName("")
}

// JustAllAttributesWithName behaviour is the same as JustAllAttributes with a filtered slice.
func (mb MergedBodies) JustAllAttributesWithName(name string) []hcl.Attributes {
	var result []hcl.Attributes

	for _, body := range mb {
		attrs, _ := body.JustAttributes()
		for attrName := range attrs {
			if name != "" && attrName != name {
				delete(attrs, attrName)
			}
		}
		if len(attrs) > 0 {
			result = append(result, attrs)
		}
	}
	return result
}

func mergeAttributes(left, right hcl.Attributes) (diags hcl.Diagnostics) {
	for name, attr := range right {
		if existing := left[name]; reportDuplicates && existing != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Duplicate argument",
				Detail: fmt.Sprintf(
					"Argument %q was already set at %s",
					name, existing.NameRange.String(),
				),
				Subject: &attr.NameRange,
			})
			continue
		}
		left[name] = attr
	}
	return diags
}
