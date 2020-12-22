package tblfmt

import (
	"reflect"
	"testing"
)

func TestFromMap(t *testing.T) {
	tests := map[string]struct {
		name       string
		opts       map[string]string
		expBuilder Builder
	}{
		"empty": {
			expBuilder: newErrEncoder,
		},
		"default format": {
			opts: map[string]string{
				"format": "aligned",
			},
			expBuilder: NewTableEncoder,
		},
		"csv format": {
			opts: map[string]string{
				"format": "csv",
			},
			expBuilder: NewCSVEncoder,
		},
		"all": {
			opts: map[string]string{
				"format":   "aligned",
				"border":   "2",
				"title":    "some title",
				"expanded": "on",
			},
			expBuilder: NewExpandedEncoder,
		},
	}

	for n, test := range tests {
		t.Run(n, func(t *testing.T) {
			builder, _ := FromMap(test.opts)
			if reflect.ValueOf(builder).Pointer() != reflect.ValueOf(test.expBuilder).Pointer() {
				t.Errorf("invalid builder, expected %+v, got %+v", test.expBuilder, builder)
			}
		})
	}
}
