// Generated by: main
// TypeWriter: holder
// Directive: +gen on InvoiceInner

package types

import (
	"github.com/tendermint/go-wire/data"
)

// Auto-generated adapters for happily unmarshaling interfaces
// Apache License 2.0
// Copyright (c) 2017 Ethan Frey (ethan.frey@tendermint.com)

type Invoice struct {
	InvoiceInner "json:\"unwrap\""
}

var InvoiceMapper = data.NewMapper(Invoice{})

func (h Invoice) MarshalJSON() ([]byte, error) {
	return InvoiceMapper.ToJSON(h.InvoiceInner)
}

func (h *Invoice) UnmarshalJSON(data []byte) (err error) {
	parsed, err := InvoiceMapper.FromJSON(data)
	if err == nil && parsed != nil {
		h.InvoiceInner = parsed.(InvoiceInner)
	}
	return err
}

// Unwrap recovers the concrete interface safely (regardless of levels of embeds)
func (h Invoice) Unwrap() InvoiceInner {
	hi := h.InvoiceInner
	for wrap, ok := hi.(Invoice); ok; wrap, ok = hi.(Invoice) {
		hi = wrap.InvoiceInner
	}
	return hi
}

func (h Invoice) Empty() bool {
	return h.InvoiceInner == nil
}

/*** below are bindings for each implementation ***/

func init() {
	InvoiceMapper.RegisterImplementation(&Contract{}, "contract", 0x1)
}

func (hi *Contract) Wrap() Invoice {
	return Invoice{hi}
}

func init() {
	InvoiceMapper.RegisterImplementation(&Expense{}, "expense", 0x2)
}

func (hi *Expense) Wrap() Invoice {
	return Invoice{hi}
}
