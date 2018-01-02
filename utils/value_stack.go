package utils

import (
  "log"
  "strconv"
  "strings"
  "time"

  "gopkg.in/olivere/elastic.v5"
)


const (
  NoField = "__ERR_NO_FIELD_SET__"
  GroupInitField = "__GROUP_INIT__"
)

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
}

var (
  // sentinel value marking the start of the "current" nested AND/OR clause, for stacking
  GroupInit = &Value{nil, GroupInitField, NoOp, false}
  // sentinel value for Value with as-yet-unset elastic.Query field
  NoQuery elastic.Query = nil
)

func NewValue(negate bool) *Value {
  return &Value{NoQuery, NoField, NoOp, negate}
}

type ValueStack struct {
  stack         []*Value
  Default       string
}

func (vs *ValueStack) Init(defField string) {
  vs.stack = []*Value{}
  vs.Default = defField
}

func (vs *ValueStack) Push(v *Value) {
  vs.stack = append(vs.stack, v)
}

func (vs *ValueStack) Pop() *Value {
  if vs.Empty() {
    return nil
  }

  last := len(vs.stack) - 1
  out := vs.stack[last]
  vs.stack = vs.stack[:last]

  return out
}

// manages temp value population during multi-step value parses. returns new value if no temp found.
// callers are expected to re-push values obtained here back onto the stack after use to continue process.
func (vs *ValueStack) current() *Value {
  // TODO: yuck! parse value elements in code not in grammar defs to simplify this crap & improve error msgs
  peek := len(vs.stack) - 1
  if vs.Empty() || vs.stack[peek] == GroupInit || vs.stack[peek].Q != NoQuery {
    return NewValue(false)
  }

  return vs.Pop()
}

func (vs *ValueStack) Empty() bool {
  return len(vs.stack) == 0
}

// start sentinel for parens-nested groupings of AND/OR separated query elements
// TODO: move ValueStack into parent QueryStack instances to disambiguate this
func (vs *ValueStack) StartGroup() {
  vs.Push(GroupInit)
}

// returns the group of values for this nested AND/OR block, and whether it was prefixed by NOT
func (vs *ValueStack) PopGroup() []*Value {
  out := []*Value{}
  next := vs.Pop()
  for next != nil && next.Field != GroupInitField {
    out = append(out, next)
    if vs.Empty() {
      break
    }
    next = vs.Pop()
  }

  return out
}

// first thing that happens in Term parsing (if present), so append a dummy vaue for filling in as we parse
func (vs *ValueStack) SetNegation() {
  vs.Push(NewValue(true))
}

// pop the tmp value stacked by SetNegation earlier, or produce
// new one if not - then fill in Field, replace on stack
func (vs *ValueStack) SetField(field string) {
  v := vs.current()
  v.Field = field
  vs.Push(v)
}

// pop the tmp value stacked by SetNegation and SetField, fill in range op, replace on stack
func (vs *ValueStack) SetRangeOp(rop RangeOp) {
  tmp := vs.current()
  tmp.RangeOp = rop
  vs.Push(tmp)
}

func (vs *ValueStack) Boolean(value string) {
  tmp := vs.current()

  b, err := strconv.ParseBool(value)
  if err != nil {
    log.Fatalf("[ERROR] failed to parse boolean from term %q for field %q, err=%s", value, tmp.Field, err)
  }

  tmp.Q = elastic.NewTermQuery(tmp.Field, b)
  vs.Push(tmp)
}

func (vs *ValueStack) Exists() {
  tmp := vs.current()
  tmp.Q = elastic.NewExistsQuery(tmp.Field)
  vs.Push(tmp)
}

func (vs *ValueStack) Date(filtered bool, value interface{}) {
  tmp := vs.current()

  if tmp.Field == NoField {
    tmp.Field = vs.Default
  }
  if filtered {
    tmp.Q = elastic.NewTermQuery(tmp.Field, value) // takes RFC3339 datetime in UTC as string
  } else {
    tmp.Q = elastic.NewMatchQuery(tmp.Field, value) // takes RFC3339 datetime in UTC as string
  }

  vs.Push(tmp)
}

func (vs *ValueStack) Number(filtered bool, value interface{}) {
  tmp := vs.current()

  if tmp.Field == NoField {
    tmp.Field = vs.Default
  }
  // drop value into "match" clause for queries, "term" clause in filter context
  if filtered {
    tmp.Q = elastic.NewTermQuery(tmp.Field, value)
  } else {
    tmp.Q = elastic.NewMatchQuery(tmp.Field, value)
  }

  vs.Push(tmp)
}

func (vs *ValueStack) Term(term interface{}) {
  tmp := vs.current()
  if tmp.Field == NoField {
    tmp.Field = vs.Default
  }
  tmp.Q = elastic.NewTermQuery(tmp.Field, term)
  vs.Push(tmp)
}

// only used in single-value context (i.e. a not a KV)
func (vs *ValueStack) Match(text interface{}) {
  tmp := vs.current()
  if tmp.Field == NoField {
    tmp.Field = vs.Default
  }
  tmp.Q = elastic.NewMatchQuery(tmp.Field, text)
  vs.Push(tmp)
}

// only used in single-value (quoted phrase) context (i.e. not a KV)
func (vs *ValueStack) Phrase(phrase string) {
  tmp := vs.current()
  if tmp.Field == NoField {
    tmp.Field = vs.Default
  }

  tmp.Q = elastic.NewMatchPhraseQuery(tmp.Field, phrase)
  vs.Push(tmp)
}

// TODO: this is hacky, separate out the number and date range handling
func (vs *ValueStack) Window(fromTildaTo string) {
  tmp := vs.current()

  fromTo := strings.Split(fromTildaTo, "~")
  var from interface{} = fromTo[0]
  var to   interface{} = fromTo[1]
  var err error
  if _, err = time.Parse(time.RFC3339, fromTo[0]); err != nil {
    if from, err = strconv.ParseFloat(fromTo[0], 10); err != nil {
      log.Fatalf("[ERROR] failed to parse range window, from args must be valid RFC3339 datetime or number, got %q, err=%s", fromTildaTo, err)
    }
  }

  if _, err = time.Parse(time.RFC3339, fromTo[1]); err != nil {
    if to, err = strconv.ParseFloat(fromTo[1], 10); err != nil {
      log.Fatalf("[ERROR] failed to parse range window, to args must be valid RFC3339 datetime or number, got %q, err=%s", fromTildaTo, err)
    }
  }

  tmp.Q = elastic.NewRangeQuery(tmp.Field).From(from).To(to).IncludeLower(true).IncludeUpper(false)
  vs.Push(tmp)
}

func (vs *ValueStack) NumberRangeOrMatchTerm(filtered bool, value string) {
  num, err := strconv.ParseFloat(value, 10)
  if err != nil {
    log.Fatalf("[ERROR] failed to parse numerical value from %q, err=%s", value, err)
  }

  // if this isn't an in-progress KV parse of a range, its a number, just pass the value along
  switch vs.Empty() || vs.stack[len(vs.stack) - 1].RangeOp == NoOp {
  case true:
    vs.Number(filtered, num)
  case false:
    vs.Range(num)
  }
}

func (vs *ValueStack) DateRangeOrMatchTerm(filtered bool, value string) {
  t, err := time.Parse(time.RFC3339, value)
  if err != nil {
    log.Fatalf("[ERROR] failed to parse RFC3339 datetime in UTC from %q, err=%s", value, err)
  }

  // if this isn't an in-progress KV parse of a range, its a number, just pass the value along
  switch vs.Empty() || vs.stack[len(vs.stack) - 1].RangeOp == NoOp {
  case true:
    vs.Date(filtered, t)
  case false:
    vs.Range(t)
  }
}

// values should land in a "match" clause in query context, "term" clause in filter context
func (vs *ValueStack) MatchTerm(filtered bool, value string) {
  switch filtered {
  case true:
    vs.Term(value)
  case false:
    vs.Match(value)
  }
}

func (vs *ValueStack) Range(value interface{}) {
  tmp := vs.current()
  rq := elastic.NewRangeQuery(tmp.Field)

  switch tmp.RangeOp {
  case LessThan:
    rq.Lt(value)
  case LessThanEqual:
    rq.Lte(value)
  case GreaterThan:
    rq.Gt(value)
  case GreaterThanEqual:
    rq.Gte(value)
  default:
    log.Fatalf("[ERROR] invalid range operation (code %d) parsing range value %q for field %q", tmp.RangeOp, value, tmp.Field)
  }
  tmp.Q = rq

  vs.Push(tmp)
}

