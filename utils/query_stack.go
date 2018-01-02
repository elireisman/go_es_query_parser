package utils

import (
  "log"

  "gopkg.in/olivere/elastic.v5"
)

type Oper uint8
const (
  Unset      Oper = iota
  DefaultAnd
  DefaultOr
  And
  Or
)

func (o Oper) String() string {
  switch o {
  case DefaultAnd: return "DEFAULT_AND"
  case DefaultOr:  return "DEFAULT_OR"
  case And:        return "AND"
  case Or:         return "OR"
  default:         return "UNSET"
  }
}

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
  if q.Oper == Unset || q.Oper == DefaultAnd || q.Oper == DefaultOr {
    q.Oper = op
  } else if q.Oper != op {
    log.Fatalf("[ERROR] mixing operators in the same query clause is illegal (current:%s, attempted:%s)", q.Oper, op)
  }
}


type QueryStack struct {
  Output        *elastic.BoolQuery
  defaultOp     Oper
  stack         []*Query
}

func NewLevel(op Oper, negate bool) *Query {
  return &Query{elastic.NewBoolQuery(), op, negate}
}

func (qs *QueryStack) Init(defaultToOr bool) {
  if defaultToOr {
    qs.defaultOp = DefaultOr
  } else {
    qs.defaultOp = DefaultAnd
  }
  qs.stack = []*Query{NewLevel(qs.defaultOp, false)}
}

func (qs *QueryStack) Empty() bool {
  return len(qs.stack) == 0
}

// obtain a pointer to the "current" query
func (qs *QueryStack) Current() *Query {
  if qs.Empty() {
    log.Fatal("[ERROR] can't manipulate current query group - the stack is empty")
  }
  return qs.stack[len(qs.stack) - 1]
}

func (qs *QueryStack) Push(negate bool) {
  qs.stack = append(qs.stack, NewLevel(qs.defaultOp, negate))
}

func (qs *QueryStack) Finalize(values []*Value) {
  result := qs.Compose(values)

  if len(qs.stack) > 1 {
    log.Println("[ERROR] input was not fully parsed, additional AST nodes remain on stack:")
    for ndx, frame := range qs.stack {
      log.Printf("[ERROR] [Stack Frame %d] %#v", len(qs.stack) - ndx, *frame)
    }
    log.Fatal("[ERROR] aborting")
  }

  // expose top-level parent ES query from final stack frame, this is our final parse result
  qs.Output = result.BoolQ
}

// when ')' or end-of-input is encountered, we pop the whole group of individual queries from the stack
// back to the last '(' or start-of-input, and we inject into the parent bool query at proper bucket/nesting
func (qs *QueryStack) Compose(values []*Value) *Query {
  for _, v := range values {
    switch qs.Current().Oper {
    // AND clause maps to Must, NOT AND to MustNot in parent query
    case And, DefaultAnd:
      if v.Negate {
        qs.Current().MustNot(v.Q)
      } else {
        qs.Current().Must(v.Q)
      }

    // OR clause maps to Should, NOT OR clause we fake w/MustNot wrapped in the parent Should
    case Or, DefaultOr:
      if v.Negate {
        qs.Current().Should(elastic.NewBoolQuery().MustNot(v.Q))
      } else {
        qs.Current().Should(v.Q)
      }

    default:
      log.Fatalf("[ERROR] invalid query clause operator in traversal results: %s", qs.Current().Oper)
    }
  }

  return qs.Pop()
}

func (qs *QueryStack) Pop() *Query {
  // pop current nested query level from stack
  if qs.Empty() {
    log.Fatal("[ERROR] can't pop subquery from empty stack, aborting")
  }
  last := len(qs.stack) - 1
  out := qs.stack[last]
  qs.stack = qs.stack[:last]

  // if this is a child (nested) subquery, nest it properly in the parent level
  if len(qs.stack) > 0 {
    switch qs.Current().Oper {
    case And, DefaultAnd:
      if out.Negate {
        // !AND: nest child query in parent's "must not"
        qs.Current().MustNot(out.BoolQ)
      } else {
        // AND: nest child query in parent's "must"
        qs.Current().Must(out.BoolQ)
      }

    case Or, DefaultOr:
      if out.Negate {
        // !OR: nest child query in "should" inside parent's "must not"
        qs.Current().MustNot(elastic.NewBoolQuery().Should(out.BoolQ))
      } else {
        // OR: nest child query in parent's "should"
        qs.Current().Should(out.BoolQ)
      }
    }
  } else {
    // restack "out" if this is the base level query, as there could
    // be multiple visits to that level before end-of-input
    qs.stack = append(qs.stack, out)
  }

  return out
}

