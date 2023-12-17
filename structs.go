package main

type Currency struct {
  Min int
  Max int
}

type Initialize struct {
  Interval int              `json:"interval"`
  Timeline []map[string]int `json:"timeline"`
}

type InitializeBase struct {
  Method   string     `json:"method"`
  Response Initialize `json:"response"`
}

type Update struct {
  Method   string            `json:"method"`
  Response [1]map[string]int `json:"response"`
}
