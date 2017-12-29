package utils

import (
  "log"

  "gopkg.in/olivere/elastic.v5"
)

type Oper uint8
const (
  None          Oper = iota
  And
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

func (q *Query) SetOper(op Oper) {
  if q.Oper != None && q.Oper != op {
    log.Fatal("a single query clause (top level or nested in parens) cannot mix AND & OR operators, aborting")
  }
  q.Oper = op
}


type QueryStack struct {
  Output        *elastic.BoolQuery
  depth         int
  stack         []*Query
}

func NewLevel(negate bool) *Query {
  return &Query{elastic.NewBoolQuery(), None, negate}
}

func (qs *QueryStack) Current() *Query {
  if qs.stack == nil {
    qs.stack = []*Query{NewLevel(false)}
  }
  return qs.stack[qs.depth]
}

func (qs *QueryStack) Push(negate bool) {
  if qs.stack == nil {
    qs.stack = []*Query{NewLevel(false)}
  }
  qs.stack = append(qs.stack, NewLevel(negate))
  qs.depth++
}

func (qs *QueryStack) Finalize(values []*Value) {
  // TODO: reuse Compose(), do final top-level checks, display remaining stack contents on fail
  // TODO: inject final qs.Pop() result into dsl.Output? return it from here?
}

func (qs *QueryStack) Compose(values []*Value) {
  // TODO: use popped value(s) and top-level group negate flag to populate qs.Current(), then qs.Pop() to nest properly
  // TODO: IF len(values) == 1, qs.SetOper(And) TO RESPECT DEFAULT
}

func (qs *QueryStack) Pop() *Query {
  out := qs.stack[qs.depth]
  qs.stack = qs.stack[:qs.depth]
  qs.depth--

  // if this is a child (nested) subquery, nest it properly in the parent level
  if qs.depth > 0 {
    switch qs.Current().Oper {
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
      log.Fatalf("unknown grouping operator type at parent level (code %d)", qs.Current().Oper)
    }
  }

  return out
}

