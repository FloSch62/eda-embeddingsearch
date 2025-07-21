package models

import (
	"fmt"
	"strings"
)

// String returns the string representation of an EQL query
func (q *EQLQuery) String() string {
	query := q.Table

	if len(q.Fields) > 0 {
		query += fmt.Sprintf(" fields [%s]", strings.Join(q.Fields, ", "))
	}

	if q.WhereClause != "" {
		query += " where (" + q.WhereClause + ")"
	}

	if len(q.OrderBy) > 0 {
		orderParts := make([]string, 0, len(q.OrderBy))
		for _, ob := range q.OrderBy {
			part := ob.Field + " " + ob.Direction
			if ob.Algorithm != "" {
				part += " " + ob.Algorithm
			}
			orderParts = append(orderParts, part)
		}
		query += fmt.Sprintf(" order by [%s]", strings.Join(orderParts, ", "))
	}

	if q.Limit > 0 {
		query += fmt.Sprintf(" limit %d", q.Limit)
	}

	if q.Delta != nil {
		query += fmt.Sprintf(" delta %s %d", q.Delta.Unit, q.Delta.Value)
	}

	return query
}
