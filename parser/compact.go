package parser

type CompactEntry struct {
	FileName string `json:"file_name"`
}

func MakeCompactEntry(entry *PakEntrySet) *CompactEntry {
	return &CompactEntry{
		FileName: entry.Summary.Record.FileName,
	}
}
