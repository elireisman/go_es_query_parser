package utils

import (
  "log"
  "regexp"
  "strconv"

  "gopkg.in/olivere/elastic"
)

var (
  // TODO: real date parsing in real fmt!!!
  SimpleDate = regexp.MustCompile(`\d{4}/\d{2}/\d{2}`)
)

type RangeOp uint8
const (
  LessThan      RangeOp = iota
  LessThanEqual
  GreaterThan
  GreaterThanEqual
)

type Value struct {
  Q             elastic.Query
  Negate        bool
  GroupStart    bool
}

type ValueStack struct {
  depth int
  stack []*Value
}

func (vs *ValueStack) doPush(v *Value) {
  if vs.stack == nil {
    vs.stack = []*Value{}
  }
  vs.stack = append(vs.stack, v)
  vs.depth++
}

func (vs *ValueStack) Push(eq elastic.Query, neg bool) {
  vs.doPush(&Value{eq, neg, false})
}

func (vs *ValueStack) Pop() *Value {
  if vs.Empty() {
    log.Fatal("invalid attempt to pop value from empty stack!")
  }

  out := vs.stack[vs.depth]
  vs.stack = vs.stack[:vs.depth]
  vs.depth--

  return out
}

// start sentinel for parens-nested groupings of AND/OR separated query elements
func (vs *ValueStack) StartGroup() {
  vs.doPush(&Value{nil, false, true})
}

func (vs *ValueStack) Empty() bool {
  return len(vs.stack) == 0
}

func (vs *ValueStack) PopGroup() []*Value {
  out := []*Value{}

  next := vs.Pop();
  for !next.GroupStart && !vs.Empty() {
    out = append(out, next)
    next = vs.Pop()
  }

  return out
}

func (vs *ValueStack) Boolean(field, value string, neg bool) {
  b, err := strconv.ParseBool(value)
  if err != nil {
    log.Fatalf("failed to parse boolean from term %q for field %q, err=%s", value, field, err)
  }
  if neg {
    b = !b
  }

  vs.Push(elastic.NewTermQuery(field, b), false)
}

func (vs *ValueStack) Exists(field string, neg bool) {
  vs.Push(elastic.NewExistsQuery(field), neg)
}

func (vs *ValueStack) Number(field, value string, neg bool) {
  i, err := strconv.Atoi(value)
  if err != nil {
    log.Fatalf("failed to parse invalid integer from term %q for field %q, err=%s", value, field, err)
  }

  vs.Push(elastic.NewTermQuery(field, i), neg)
}

func (vs *ValueStack) Term(field, term string, neg bool) {
  vs.Push(elastic.NewTermQuery(field, term), neg)
}

// value accepts dates "YYYY/MM/DD" or integers "-93"
func (vs *ValueStack) Range(field, value string, op RangeOp, neg bool) {
  rq := elastic.NewRangeQuery(field)

  var v interface{}
  var err error
  if SimpleDate.MatchString(value) {
    v = value
  } else if v, err = strconv.Atoi(value); err != nil {
    log.Fatalf("couldn't parse valid integer for range value %q for field %q, err=%s", value, field, err)
  }

  switch op {
  case LessThan:
    rq.Lt(v)
  case LessThanEqual:
    rq.Lte(v)
  case GreaterThan:
    rq.Gt(v)
  case GreaterThanEqual:
    rq.Gte(v)
  default:
    log.Fatalf("invalid range operation (code %d) parsing range value %q for field %q", op, value, field)
  }

  vs.Push(rq, neg)
}

func (vs *ValueStack) Match(field, text string, neg bool) {
  vs.Push(elastic.NewMatchQuery(field, text), neg)
}

func (vs *ValueStack) Phrase(field, phrase string, neg bool) {
  vs.Push(elastic.NewMatchPhraseQuery(field, phrase), neg)
}

