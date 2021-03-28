package tblfmt

import (
	"reflect"
	"testing"
)

func TestFromMap(t *testing.T) {
	tests := map[string]struct {
		name string
		opts map[string]string
		exp  Builder
	}{
		"empty": {
			exp: newErrEncoder,
		},
		"aligned": {
			opts: map[string]string{
				"format": "aligned",
			},
			exp: NewTableEncoder,
		},
		"unaligned": {
			opts: map[string]string{
				"format": "unaligned",
			},
			exp: NewUnalignedEncoder,
		},
		"csv": {
			opts: map[string]string{
				"format": "csv",
			},
			exp: NewUnalignedEncoder,
		},
		"any": {
			opts: map[string]string{
				"format":   "aligned",
				"border":   "2",
				"title":    "some title",
				"expanded": "on",
			},
			exp: NewExpandedEncoder,
		},
	}
	for n, test := range tests {
		t.Run(n, func(t *testing.T) {
			builder, _ := FromMap(test.opts)
			if reflect.ValueOf(builder).Pointer() != reflect.ValueOf(test.exp).Pointer() {
				t.Errorf("invalid builder, expected %T, got %T", test.exp, builder)
			}
		})
	}
}
