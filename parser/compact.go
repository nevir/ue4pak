package parser

type CompactEntry struct {
	// TODO: Figure out how to reference this from Names.
	// Package  string           `json:"package"`
	FileName string           `json:"file_name"`
	GUID     *FGuid           `json:"guid"`
	Exports  []*CompactExport `json:"exports"`
}

type CompactExport struct {
	Name       string                      `json:"name"`
	Class      *CompactReference           `json:"class"`
	Super      *CompactReference           `json:"super"`
	Template   *CompactReference           `json:"template"`
	Outer      *CompactReference           `json:"outer"`
	Properties map[string]*CompactProperty `json:"properties"`
}

type CompactReference struct {
	Package string `json:"package"`
	Name    string `json:"name"`
}

type CompactProperty struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func MakeCompactEntry(entry *PakEntrySet) *CompactEntry {
	exports := make([]*CompactExport, len(entry.Exports))
	for i, export := range entry.Exports {
		exports[i] = MakeCompactExport(&export)
	}

	return &CompactEntry{
		FileName: entry.Summary.Record.FileName,
		GUID:     entry.Summary.GUID,
		Exports:  exports,
	}
}

func MakeCompactExport(export *PakExportSet) *CompactExport {
	return &CompactExport{
		Name:       export.Export.ObjectName,
		Class:      MakeCompactReference(export.Export.ClassIndex),
		Super:      MakeCompactReference(export.Export.SuperIndex),
		Template:   MakeCompactReference(export.Export.TemplateIndex),
		Outer:      MakeCompactReference(export.Export.OuterIndex),
		Properties: MakePropertyMap(export.Properties),
	}
}

func MakeCompactReference(index *FPackageIndex) *CompactReference {
	if index.Reference == nil {
		return nil
	}

	imported, ok := index.Reference.(*FObjectImport)
	if ok {
		outerPackage := imported.OuterPackage.Reference.(*FObjectImport)

		return &CompactReference{
			Package: outerPackage.ObjectName,
			Name:    imported.ObjectName,
		}
	}

	exported, ok := index.Reference.(*FObjectExport)
	if ok {
		// TODO: Is this a bug?
		if exported == nil {
			return &CompactReference{
				Package: "{{THIS PACKAGE}}",
				Name:    "{{BROKEN/MISSING EXPORT}}",
			}
		}

		return &CompactReference{
			// It shows up in Names, but isn't referenced by our structs.
			// TODO: Figure out how to extract it.
			Package: "{{THIS PACKAGE}}",
			Name:    exported.ObjectName,
		}
	}

	panic("Unknown reference type")
}

func MakePropertyMap(tags []*FPropertyTag) map[string]*CompactProperty {
	properties := make(map[string]*CompactProperty)
	for _, property := range tags {
		properties[property.Name] = MakeCompactProperty(property)
	}

	return properties
}

func MakeCompactProperty(tag *FPropertyTag) *CompactProperty {
	simpleStruct, ok := tag.Tag.(*StructType)
	if ok {
		return &CompactProperty{
			Type:  simpleStruct.Type,
			Value: simpleStruct.Value,
		}
	}

	if tag.PropertyType == "BoolProperty" {
		return &CompactProperty{
			Type:  tag.PropertyType,
			Value: tag.TagData,
		}
	}

	if tag.PropertyType == "EnumProperty" {
		return &CompactProperty{
			Type:  tag.TagData.(string),
			Value: tag.Tag,
		}
	}

	if tag.PropertyType == "StructProperty" {
		structTag := tag.TagData.(*StructProperty)

		var properties map[string]*CompactProperty
		subTags, ok := tag.Tag.([]*FPropertyTag)
		if ok {
			properties = MakePropertyMap(subTags)
		}

		return &CompactProperty{
			Type:  structTag.Type,
			Value: properties,
		}
	}

	return &CompactProperty{
		Type:  tag.PropertyType,
		Value: MakePropertyValue(tag.Tag),
	}
}

func MakePropertyValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	array, ok := value.([]interface{})
	if ok {
		values := make([]interface{}, len(array))
		for i, value := range array {
			values[i] = MakePropertyValue(value)
		}
		return values
	}

	arrayStruct, ok := value.(*ArrayStructProperty)
	if ok {
		simpleStruct, ok := arrayStruct.Properties.(*StructType)
		if ok {
			return simpleStruct
		}

		structData := arrayStruct.InnerTagData.TagData.(*StructProperty)
		return &CompactProperty{
			Type:  structData.Type,
			Value: MakePropertyMap(arrayStruct.Properties.([]*FPropertyTag)),
		}
	}

	reference, ok := value.(*FPackageIndex)
	if ok {
		return &CompactProperty{
			Type:  "reference",
			Value: MakeCompactReference(reference),
		}
	}

	return value
}
