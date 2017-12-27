package utils

import (
  "log"

  "gopkg.in/olivere/elastic"
)

type Oper uint8
const (
  And           Oper = iota
  Or
)

type Query struct {
  BoolQ         *elastic.BoolQuery
  Oper          Oper
  Negate        bool
}

func (q *Query) Must(eq elastic.Query) {
  q.BoolQ.Must(eq)
}

func (q *Query) MustNot(eq elastic.Query) {
  q.BoolQ.MustNot(eq)
}

func (q *Query) Should(eq elastic.Query) {
  q.BoolQ.Should(eq)
}

type QueryStack struct {
  depth         int
  stack         []*Query
}

func NewLevel() *Query {
  return &Query{elastic.NewBoolQuery(), And, false}
}

func (qs *QueryStack) Current() *Query {
  if qs.stack == nil {
    qs.stack = []*Query{NewLevel()}
  }
  return qs.stack[qs.depth]
}

func (qs *QueryStack) Push(b Oper) {
  if qs.stack == nil {
    qs.stack = []*Query{NewLevel()}
  }
  qs.stack = append(qs.stack, NewLevel())
  qs.depth++
}

func (qs *QueryStack) Pop() *Query {
  out := qs.stack[qs.depth]
  qs.stack = qs.stack[:qs.depth]
  qs.depth--

  // if this is a child (nested) subquery, nest it properly in the parent level
  if qs.depth > 0 {
    switch out.Oper {
    case And:
      if out.Negate {
        // !AND: nest child query in parent's "must not"
        qs.Current().MustNot(out.BoolQ)
      } else {
        // AND: nest child query in parent's "must"
        qs.Current().Must(out.BoolQ)
      }

    case Or:
      if out.Negate {
        // !OR: nest child query in "should" inside parent's "must not"
        qs.Current().MustNot(elastic.NewBoolQuery().Should(out.BoolQ))
      } else {
        // OR: nest child query in parent's "should"
        qs.Current().Should(out.BoolQ)
      }

    default:
      log.Fatalf("unknown grouping operator type (code %d)", out.Oper)
    }
  }

  return out
}

