package ports

import "testing"

func TestFileSystemOp_String(t *testing.T) {
	tests := []struct {
		op       FileSystemOp
		expected string
	}{
		{OpCreate, "create"},
		{OpRemove, "remove"},
		{OpRename, "rename"},
		{OpModify, "modify"},
		{FileSystemOp(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.op.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
