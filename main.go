package main


import (
  "fmt"
  "flag"
  "log"
  "os"

  "github.com/elireisman/go_es_query_parse/grammar"
)

const NoInput = "ERR_NO_INPUT_PROVIDED"

func main() {
  query := flag.String("query", NoInput, "the query (written in the DSL) you wish to submit")
  verbose := flag.Bool("verbose", false, "log/explain verbosely during parsing")
  halp := flag.Bool("help", false, "print DSL and usage details and exit")
  flag.Parse()

  if *query == NoInput {
    log.Println("-query argument specifying query string is required, aborting!")
    os.Exit(1)
  }
  if *halp {
    log.Println(usage())
    os.Exit(1)
  }

  log.Printf("Query: %q\t\t(verbose=%t, help=%t)", *query, *verbose, *halp)

  dsl := &grammar.DSL2ES{Buffer: *query}
  dsl.Init()
  if err := dsl.Parse(); err != nil {
    log.Fatalf("Parse Error: %s", err)
  }
}

func usage() string {
  return fmt.Sprintf("Usage: %s -query \"query string\" [-verbose] [-help]", os.Args[0])

  // TODO: detail the DSL grammar etc. here also, or with verbose + help opts together only?
}
