// Code generated by "stringer -output=zfieldtype_strings.go -type=fwpmFieldType -trimprefix=fwpmFieldtype"; DO NOT EDIT.

package wf

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[fwpmFieldTypeRawData-0]
	_ = x[fwpmFieldTypeIPAddress-1]
	_ = x[fwpmFieldTypeFlags-2]
}

const _fwpmFieldType_name = "fwpmFieldTypeRawDatafwpmFieldTypeIPAddressfwpmFieldTypeFlags"

var _fwpmFieldType_index = [...]uint8{0, 20, 42, 60}

func (i fwpmFieldType) String() string {
	if i >= fwpmFieldType(len(_fwpmFieldType_index)-1) {
		return "fwpmFieldType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _fwpmFieldType_name[_fwpmFieldType_index[i]:_fwpmFieldType_index[i+1]]
}