// Package tblfmt provides encoders for streaming / writing rows of data.
package tblfmt

import (
	"io"
)

// EncodeTable encodes result set to the writer using the supplied encoding
// options.
func EncodeTable(w io.Writer, rs ResultSet, opts ...TableEncoderOption) error {
	enc, err := NewTableEncoder(rs, opts...)
	if err != nil {
		return err
	}
	return enc.Encode(w)
}

// EncodeTableAll encodes all result sets to the writer using the supplied
// encoding options.
func EncodeTableAll(w io.Writer, rs ResultSet, opts ...TableEncoderOption) error {
	enc, err := NewTableEncoder(rs, opts...)
	if err != nil {
		return err
	}
	return enc.EncodeAll(w)
}
