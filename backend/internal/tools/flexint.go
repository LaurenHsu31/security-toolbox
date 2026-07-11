package tools

import (
	"fmt"
	"strconv"
	"strings"
)

// flexInt is an int that also accepts JSON strings ("42", "") and null.
// The frontend sends every control value as a string, so plain int fields
// would reject e.g. {"tagLen": ""} and make the whole tool unusable.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(strings.Trim(string(b), `"`))
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("%q is not a whole number", s)
	}
	*f = flexInt(n)
	return nil
}
