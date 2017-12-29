package utils

import (
  "log"
  "regexp"
  "strconv"

  "gopkg.in/olivere/elastic.v5"
)

var (
  // TODO: real date parsing in real fmt!!!
  SimpleDate = regexp.MustCompile(`\d{4}/\d{2}/\d{2}`)
)

const NoField = "__ERR_NO_FIELD_SET__"

type RangeOp uint8
const (
  NoOp          RangeOp = iota
  LessThan
  LessThanEqual
  GreaterThan
  GreaterThanEqual
)

type Value struct {
  Q             elastic.Query
  Field         string
  RangeOp       RangeOp
  Negate        bool
  GroupStart    bool
}

type ValueStack struct {
  stack []*Value
}

func (vs *ValueStack) Peek() *Value {
  if vs.Empty() {
    return nil
  }
  return vs.stack[len(vs.stack) - 1]
}

// first thing that happens in Term parsing, so append a dummy vaue for filling in as we parse
func (vs *ValueStack) SetNegation(neg bool) {
  vs.Push(&Value{nil, NoField, NoOp, neg, false})
}

// pop the tmp value stacked by SetNegation earlier, fill in Field, replace on stack
func (vs *ValueStack) SetField(field string) {
  v := vs.Pop()
  v.Field = field
  vs.Push(v)
}

// pop the tmp value stacked by SetNegation and SetField, fill in range op, replace on stack
func (vs *ValueStack) SetRangeOp(rop string) {
  v := vs.Pop()
  if v.Field == NoField {
    log.Fatalf("failed to register range operator %q, no field present to apply it to, aborting", rop)
  }

  switch rop {
  case ">=":
    v.RangeOp = GreaterThanEqual
  case "<=":
    v.RangeOp = LessThanEqual
  case ">":
    v.RangeOp = GreaterThan
  case "<":
    v.RangeOp = LessThan
  default:
    log.Fatalf("invalid range operator %q found, aborting", rop)
  }

  vs.Push(v)
}

func (vs *ValueStack) Push(v *Value) {
  if vs.stack == nil {
    vs.stack = []*Value{}
  }
  vs.stack = append(vs.stack, v)
}

func (vs *ValueStack) Pop() *Value {
  if vs.Empty() {
    log.Fatal("invalid attempt to pop value from empty stack!")
  }

  last := len(vs.stack) - 1
  out := vs.stack[last]
  vs.stack = vs.stack[:last]

  return out
}

func (vs *ValueStack) Empty() bool {
  return len(vs.stack) == 0
}

// start sentinel for parens-nested groupings of AND/OR separated query elements
func (vs *ValueStack) StartGroup() {
  vs.Push(&Value{nil, NoField, NoOp, false, true})
}

// returns the group of values for this nested AND/OR block, and whether it was prefixed by NOT
func (vs *ValueStack) PopGroup() []*Value {
  out := []*Value{}

  // capture GroupStart too, strip it and return the negation value
  next := vs.Pop();
  for !next.GroupStart && !vs.Empty() {
    out = append(out, next)
    next = vs.Pop()
  }

  if len(out) > 0 && out[0].GroupStart {
    out = out[1:]
  }
  return out
}

func (vs *ValueStack) Boolean(value string) {
  tmp := vs.Pop()

  b, err := strconv.ParseBool(value)
  if err != nil {
    log.Fatalf("failed to parse boolean from term %q for field %q, err=%s", value, tmp.Field, err)
  }

  tmp.Q = elastic.NewTermQuery(tmp.Field, b)
  vs.Push(tmp)
}

func (vs *ValueStack) Exists() {
  tmp := vs.Pop()
  tmp.Q = elastic.NewExistsQuery(tmp.Field)
  vs.Push(tmp)
}

func (vs *ValueStack) Number(value string) {
  tmp := vs.Pop()

  i, err := strconv.Atoi(value)
  if err != nil {
    log.Fatalf("failed to parse invalid integer from term %q for field %q, err=%s", value, tmp.Field, err)
  }

  tmp.Q = elastic.NewTermQuery(tmp.Field, i)
  vs.Push(tmp)
}

func (vs *ValueStack) Term(term string) {
  tmp := vs.Pop()
  tmp.Q = elastic.NewTermQuery(tmp.Field, term)
  vs.Push(tmp)
}

// only used in single-value context (i.e. a not a KV)
func (vs *ValueStack) Match(text string, isFilterCtx bool) {
  if isFilterCtx {
    log.Fatal("no match query clauses allowed in filter context (try removing --filter CLI arg)")
  }
  tmp := vs.Pop()
  tmp.Q = elastic.NewMatchQuery(tmp.Field, text)
  vs.Push(tmp)
}

// only used in single-value (quoted phrase) context (i.e. not a KV)
func (vs *ValueStack) Phrase(phrase string, isFilterCtx bool) {
  if isFilterCtx {
    log.Fatal("no match_phrase query clauses allowed in filter context (try removing --filter CLI arg)")
  }
  tmp := vs.Pop()
  tmp.Q = elastic.NewMatchPhraseQuery(tmp.Field, phrase)
  vs.Push(tmp)
}

// value accepts dates "YYYY/MM/DD" or integers "-93"
func (vs *ValueStack) Range(value string) {
  tmp := vs.Pop()

  if vs.Peek() != nil && vs.Peek().RangeOp == NoOp {
    log.Fatal("range op expected at top of values stack, can't register value %q for field %q, aborting", value, tmp.Field)
  }
  rq := elastic.NewRangeQuery(tmp.Field)

  var v interface{}
  var err error
  if SimpleDate.MatchString(value) {
    v = value
  } else if v, err = strconv.Atoi(value); err != nil {
    log.Fatalf("couldn't parse valid integer for range value %q for field %q, err=%s", value, tmp.Field, err)
  }

  switch tmp.RangeOp {
  case LessThan:
    rq.Lt(v)
  case LessThanEqual:
    rq.Lte(v)
  case GreaterThan:
    rq.Gt(v)
  case GreaterThanEqual:
    rq.Gte(v)
  default:
    log.Fatalf("invalid range operation (code %d) parsing range value %q for field %q", tmp.RangeOp, value, tmp.Field)
  }
  tmp.Q = rq

  vs.Push(tmp)
}

