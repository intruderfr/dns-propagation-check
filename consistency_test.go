package main

import "testing"

func TestIsConsistent_AllAgree(t *testing.T) {
	r := []Result{
		{Resolver: Resolver{Provider: "A"}, Answers: []string{"1.2.3.4"}},
		{Resolver: Resolver{Provider: "B"}, Answers: []string{"1.2.3.4"}},
	}
	if !isConsistent(r) {
		t.Fatal("expected consistent, got inconsistent")
	}
}

func TestIsConsistent_Disagree(t *testing.T) {
	r := []Result{
		{Resolver: Resolver{Provider: "A"}, Answers: []string{"1.2.3.4"}},
		{Resolver: Resolver{Provider: "B"}, Answers: []string{"5.6.7.8"}},
	}
	if isConsistent(r) {
		t.Fatal("expected inconsistent, got consistent")
	}
}

func TestIsConsistent_ErrorsIgnored(t *testing.T) {
	r := []Result{
		{Resolver: Resolver{Provider: "A"}, Answers: []string{"1.2.3.4"}},
		{Resolver: Resolver{Provider: "B"}, Err: errStub("nxdomain")},
		{Resolver: Resolver{Provider: "C"}, Answers: []string{"1.2.3.4"}},
	}
	if !isConsistent(r) {
		t.Fatal("errored resolvers should be ignored when checking consistency")
	}
}

func TestIsConsistent_AllErrors(t *testing.T) {
	r := []Result{
		{Resolver: Resolver{Provider: "A"}, Err: errStub("x")},
		{Resolver: Resolver{Provider: "B"}, Err: errStub("y")},
	}
	if isConsistent(r) {
		t.Fatal("no successful results — should not be considered consistent")
	}
}

func TestGroupByAnswer(t *testing.T) {
	r := []Result{
		{Resolver: Resolver{Provider: "A"}, Answers: []string{"1.2.3.4"}},
		{Resolver: Resolver{Provider: "B"}, Answers: []string{"5.6.7.8"}},
		{Resolver: Resolver{Provider: "C"}, Answers: []string{"1.2.3.4"}},
	}
	groups := groupByAnswer(r)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if len(groups["1.2.3.4"]) != 2 {
		t.Fatalf("expected 2 resolvers in group 1.2.3.4, got %d", len(groups["1.2.3.4"]))
	}
}

func TestParseCustomResolvers(t *testing.T) {
	got := parseCustomResolvers("8.8.8.8, 1.1.1.1 , , 9.9.9.9")
	if len(got) != 3 {
		t.Fatalf("expected 3 resolvers, got %d", len(got))
	}
	if got[0].IP != "8.8.8.8" || got[2].IP != "9.9.9.9" {
		t.Fatalf("unexpected parse result: %+v", got)
	}
}

type errStub string

func (e errStub) Error() string { return string(e) }
