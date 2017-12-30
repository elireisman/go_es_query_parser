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
  stack         []*Query
}

func NewLevel(negate bool) *Query {
  return &Query{elastic.NewBoolQuery(), None, negate}
}

func (qs *QueryStack) Init() {
  qs.stack = []*Query{NewLevel(false)}
}

func (qs *QueryStack) Current() *Query {
  return qs.stack[len(qs.stack) - 1]
}

func (qs *QueryStack) Push(negate bool) {
  qs.stack = append(qs.stack, NewLevel(negate))
}

func (qs *QueryStack) Finalize(values []*Value) {
  result := qs.Compose(values)

  if len(qs.stack) > 0 {
    log.Println("input was not fully parsed, additional AST nodes remain on stack:")
    for ndx, frame := range qs.stack {
      log.Printf("[Stack Frame %d] %#v", len(qs.stack) - ndx, *frame)
    }
    log.Fatal("Aborting.")
  }

  // expose top-level parent ES query from final stack frame, this is our final parse result
  qs.Output = result.BoolQ
}

// when ')' or end-of-input is encountered, we pop the whole group of individual queries from the stack
// back to the last '(' or start-of-input, and we inject into the parent bool query at proper bucket/nesting
func (qs *QueryStack) Compose(values []*Value) *Query {
  if len(values) == 0 {
    log.Fatal("every AND/OR clause must contain at least one valid value argument, aborting")
  } else if len(values) == 1 {
    // AND is the default for unspecified groups
    if qs.Current().Oper != And {
        qs.Current().Oper = And
    }
  }

  for ndx, v := range values {
    switch qs.Current().Oper {
    // AND clause maps to Must, NOT AND to MustNot in parent query
    case And:
      if v.Negate {
        qs.Current().MustNot(v.Q)
      } else {
        qs.Current().Must(v.Q)
      }

    // OR clause maps to Should, NOT OR clause we fake w/MustNot wrapped in the parent Should
    case Or:
      if v.Negate {
        qs.Current().Should(elastic.NewBoolQuery().MustNot(v.Q))
      } else {
        qs.Current().Should(v.Q)
      }

    default: // None
      log.Fatalf("unknown operator encountered in grouping clause at position %d, aborting", ndx)
    }
  }

  return qs.Pop()
}

func (qs *QueryStack) Pop() *Query {
  last := len(qs.stack) - 1
  out := qs.stack[last]
  qs.stack = qs.stack[:last]

  log.Fatalf("[DEBUG] qs.Pop(stack_size:%d)", len(qs.stack)) // TODO: DEBUG, REMOVE!

  // if this is a child (nested) subquery, nest it properly in the parent level
  if len(qs.stack) > 0 {
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

