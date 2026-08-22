package schema

// ValidateConnection checks whether source and target types share at least one
// common base class. Returns a ValidationResult with the overlap.
func ValidateConnection(sourceTypes, targetTypes []string) ValidationResult {
	common := FindCompatibleTypes(sourceTypes, map[string][]string{
		"source": sourceTypes,
	})

	// Check for direct intersection.
	sourceSet := make(map[string]bool, len(sourceTypes))
	for _, t := range sourceTypes {
		sourceSet[t] = true
	}

	targetSet := make(map[string]bool, len(targetTypes))
	for _, t := range targetTypes {
		targetSet[t] = true
	}

	var shared []string
	for _, t := range sourceTypes {
		if targetSet[t] {
			shared = append(shared, t)
		}
	}
	// Also check the other direction.
	for _, t := range targetTypes {
		if sourceSet[t] {
			already := false
			for _, s := range shared {
				if s == t {
					already = true
					break
				}
			}
			if !already {
				shared = append(shared, t)
			}
		}
	}

	_ = common // used by FindCompatibleTypes pattern

	if len(shared) > 0 {
		return ValidationResult{
			Valid:       true,
			Common:      shared,
			SourceTypes: sourceTypes,
			TargetTypes: targetTypes,
		}
	}

	return ValidationResult{
		Valid:       false,
		Message:     "no common types between source and target",
		SourceTypes: sourceTypes,
		TargetTypes: targetTypes,
	}
}

// FindCompatibleTypes returns the subset of sourceTypes that appear in any
// of the output type lists provided in allOutputs.
func FindCompatibleTypes(sourceTypes []string, allOutputs map[string][]string) []string {
	// Build a set of all available output types.
	available := make(map[string]bool)
	for _, types := range allOutputs {
		for _, t := range types {
			available[t] = true
		}
	}

	var compatible []string
	for _, t := range sourceTypes {
		if available[t] {
			compatible = append(compatible, t)
		}
	}
	return compatible
}

// IsFieldHidden returns true if the field's Show property is false,
// meaning the field should not be visible in the UI.
func IsFieldHidden(field TemplateField) bool {
	return !field.Show
}

// IsToolModeConflict returns true if the node has tool_mode enabled and
// the named field also has tool_mode=true. In this case, the connection
// should be removed because the Agent will supply the value at call time.
func IsToolModeConflict(node Node, fieldName string) bool {
	// Check if this node's outputs have been replaced by tool_mode.
	// When tool_mode is active, the node typically only has
	// "component_as_tool" output and base_classes become ["Tool"].
	isToolModeNode := false
	for _, bc := range node.Data.Node.BaseClasses {
		if bc == "Tool" {
			isToolModeNode = true
			break
		}
	}
	if !isToolModeNode {
		return false
	}

	field, ok := node.Data.Node.Template[fieldName]
	if !ok {
		return false
	}
	return field.ToolMode
}
