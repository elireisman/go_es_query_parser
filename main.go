package main


import (
  "fmt"
  "flag"
  "os"
)

const NoInput = "ERR_NO_INPUT_PROVIDED"

func main() {
  rawQuery := flag.String("query", NoInput, "the query (written in the DSL) you wish to submit")
  verbose := flag.Bool("verbose", false, "log/explain verbosely during parsing")
  halp := flag.Bool("help", false, "print DSL and usage details and exit")
  flag.Parse()

  fmt.Printf("%q\t(verbose=%t)\t(help=%t)", *rawQuery, *verbose, *halp)

  // TODO: create generated parser object and DO THE PARSE, print JSON to stdout if all goes well
}

func usage() string {
  return fmt.Sprintf("Usage: %s -query \"(NOT (foo OR (bar AND baz)) AND dog AND NOT (pug OR lab)\" [-verbose] [-help]", os.Args[0])

  // TODO: detail the DSL grammar etc. here also, or with verbose + help opts together only?
}
