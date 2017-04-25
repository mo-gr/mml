package mml

import (
	"bytes"
	"testing"
)

func TestParseMML(t *testing.T) {
	s, err := newMMLSyntax()
	if err != nil {
		t.Error(err)
		return
	}

	s.traceLevel = traceDebug

	for _, ti := range []struct {
		msg   string
		text  string
		nodes []*node
		fail  bool
	}{{
		msg: "empty document",
	}, {
		msg:  "single int",
		text: "42",
		nodes: []*node{{
			typeName: "int",
			token:    &token{value: "42"},
		}},
	}, {
		msg:  "multiple ints",
		text: "1 2\n3;4 ;\n 5",
		nodes: []*node{{
			typeName: "int",
			token:    &token{value: "1"},
		}, {
			typeName: "int",
			token:    &token{value: "2"},
		}, {
			typeName: "nl",
			token:    &token{value: "\n"},
		}, {
			typeName: "int",
			token:    &token{value: "3"},
		}, {
			typeName: "semicolon",
			token:    &token{value: ";"},
		}, {
			typeName: "int",
			token:    &token{value: "4"},
		}, {
			typeName: "semicolon",
			token:    &token{value: ";"},
		}, {
			typeName: "nl",
			token:    &token{value: "\n"},
		}, {
			typeName: "int",
			token:    &token{value: "5"},
		}},
	}, {
		msg:  "string",
		text: "\"foo\"",
		nodes: []*node{{
			typeName: "string",
			token:    &token{value: "\"foo\""},
		}},
	}, {
		msg:  "bool",
		text: "true false",
		nodes: []*node{{
			typeName: "true",
			token:    &token{value: "true"},
		}, {
			typeName: "false",
			token:    &token{value: "false"},
		}},
	}, {
		msg:  "symbol",
		text: "foo",
		nodes: []*node{{
			typeName: "symbol",
			token:    &token{value: "foo"},
		}},
	}, {
		msg:  "dynamic symbol",
		text: "symbol(f(a))",
		nodes: []*node{{
			typeName: "dynamic-symbol",
			token:    &token{value: "symbol"},
			nodes: []*node{{
				typeName: "symbol-word",
				token:    &token{value: "symbol"},
			}, {
				typeName: "open-paren",
				token:    &token{value: "("},
			}, {
				typeName: "nls",
				token:    &token{value: "f"},
			}, {
				typeName: "function-call",
				token:    &token{value: "f"},
				nodes: []*node{{
					typeName: "symbol",
					token:    &token{value: "f"},
				}, {
					typeName: "open-paren",
					token:    &token{value: "("},
				}, {
					typeName: "list-sequence",
					token:    &token{value: "a"},
					nodes: []*node{{
						typeName: "symbol",
						token:    &token{value: "a"},
					}},
				}, {
					typeName: "close-paren",
					token:    &token{value: ")"},
				}},
			}, {
				typeName: "nls",
				token:    &token{value: ")"},
			}, {
				typeName: "close-paren",
				token:    &token{value: ")"},
			}},
		}},
	}, {
		msg:  "empty list",
		text: "[]",
		nodes: []*node{{
			typeName: "list",
			token:    &token{value: "["},
			nodes: []*node{{
				typeName: "open-square",
				token:    &token{value: "["},
			}, {
				typeName: "list-sequence",
				token:    &token{value: "]"},
			}, {
				typeName: "close-square",
				token:    &token{value: "]"},
			}},
		}},
	}, {
		msg:  "list",
		text: "[1, 2, f(a), [3, 4, []]]",
		nodes: []*node{{
			typeName: "list",
			token:    &token{value: "["},
			nodes: []*node{{
				typeName: "open-square",
				token:    &token{value: "["},
			}, {
				typeName: "list-sequence",
				token:    &token{value: "1"},
				nodes: []*node{{
					typeName: "int",
					token:    &token{value: "1"},
				}, {
					typeName: "comma",
					token:    &token{value: ","},
				}, {
					typeName: "int",
					token:    &token{value: "2"},
				}, {
					typeName: "comma",
					token:    &token{value: ","},
				}, {
					typeName: "function-call",
					token:    &token{value: "f"},
					nodes: []*node{{
						typeName: "symbol",
						token:    &token{value: "f"},
					}, {
						typeName: "open-paren",
						token:    &token{value: "("},
					}, {
						typeName: "list-sequence",
						token:    &token{value: "a"},
						nodes: []*node{{
							typeName: "symbol",
							token:    &token{value: "a"},
						}},
					}, {
						typeName: "close-paren",
						token:    &token{value: ")"},
					}},
				}, {
					typeName: "comma",
					token:    &token{value: ","},
				}, {
					typeName: "list",
					token:    &token{value: "["},
					nodes: []*node{{
						typeName: "open-square",
						token:    &token{value: "["},
					}, {
						typeName: "list-sequence",
						token:    &token{value: "3"},
						nodes: []*node{{
							typeName: "int",
							token:    &token{value: "3"},
						}, {
							typeName: "comma",
							token:    &token{value: ","},
						}, {
							typeName: "int",
							token:    &token{value: "4"},
						}, {
							typeName: "comma",
							token:    &token{value: ","},
						}, {
							typeName: "list",
							token:    &token{value: "["},
							nodes: []*node{{
								typeName: "open-square",
								token:    &token{value: "["},
							}, {
								typeName: "list-sequence",
								token:    &token{value: "]"},
							}, {
								typeName: "close-square",
								token:    &token{value: "]"},
							}},
						}},
					}, {
						typeName: "close-square",
						token:    &token{value: "]"},
					}},
				}},
			}, {
				typeName: "close-square",
				token:    &token{value: "]"},
			}},
		}},
	}, {
		msg:  "mutable list",
		text: "~[1, 2, f(a), [3, 4, ~[]]]",
		nodes: []*node{{
			typeName: "mutable-list",
			token:    &token{value: "~"},
			nodes: []*node{{
				typeName: "tilde",
				token:    &token{value: "~"},
			}, {
				typeName: "open-square",
				token:    &token{value: "["},
			}, {
				typeName: "list-sequence",
				token:    &token{value: "1"},
				nodes: []*node{{
					typeName: "int",
					token:    &token{value: "1"},
				}, {
					typeName: "comma",
					token:    &token{value: ","},
				}, {
					typeName: "int",
					token:    &token{value: "2"},
				}, {
					typeName: "comma",
					token:    &token{value: ","},
				}, {
					typeName: "function-call",
					token:    &token{value: "f"},
					nodes: []*node{{
						typeName: "symbol",
						token:    &token{value: "f"},
					}, {
						typeName: "open-paren",
						token:    &token{value: "("},
					}, {
						typeName: "list-sequence",
						token:    &token{value: "a"},
						nodes: []*node{{
							typeName: "symbol",
							token:    &token{value: "a"},
						}},
					}, {
						typeName: "close-paren",
						token:    &token{value: ")"},
					}},
				}, {
					typeName: "comma",
					token:    &token{value: ","},
				}, {
					typeName: "list",
					token:    &token{value: "["},
					nodes: []*node{{
						typeName: "open-square",
						token:    &token{value: "["},
					}, {
						typeName: "list-sequence",
						token:    &token{value: "3"},
						nodes: []*node{{
							typeName: "int",
							token:    &token{value: "3"},
						}, {
							typeName: "comma",
							token:    &token{value: ","},
						}, {
							typeName: "int",
							token:    &token{value: "4"},
						}, {
							typeName: "comma",
							token:    &token{value: ","},
						}, {
							typeName: "mutable-list",
							token:    &token{value: "~"},
							nodes: []*node{{
								typeName: "tilde",
								token:    &token{value: "~"},
							}, {
								typeName: "open-square",
								token:    &token{value: "["},
							}, {
								typeName: "list-sequence",
								token:    &token{value: "]"},
							}, {
								typeName: "close-square",
								token:    &token{value: "]"},
							}},
						}},
					}, {
						typeName: "close-square",
						token:    &token{value: "]"},
					}},
				}},
			}, {
				typeName: "close-square",
				token:    &token{value: "]"},
			}},
		}},
	}} {
		t.Run(ti.msg, func(t *testing.T) {
			b := bytes.NewBufferString(ti.text)
			r := newTokenReader(b, "<test>")

			n, err := s.parse(r)
			if !ti.fail && err != nil {
				t.Error(err)
				return
			} else if ti.fail && err == nil {
				t.Error("failed to fail")
				return
			}

			if ti.fail {
				return
			}

			if n.typeName != "statement-sequence" {
				t.Error("invalid root node type", n.typeName, "statement-sequence")
				return
			}

			if len(n.nodes) != len(ti.nodes) {
				t.Error("invalid number of nodes", len(n.nodes), len(ti.nodes))
				return
			}

			if len(n.nodes) == 0 && n.token != eofToken || len(n.nodes) > 0 && n.token != n.nodes[0].token {
				t.Error("invalid document token", n.token)
				return
			}

			for i, ni := range n.nodes {
				if !checkNodes(ni, ti.nodes[i]) {
					t.Error("failed to match nodes")
					t.Log(ni)
					t.Log(ti.nodes[i])
				}
			}
		})
	}
}
