package next

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"time"
)

type Options struct {
	Trace Trace
}

type Syntax struct {
	trace       Trace
	registry    *registry
	initialized bool
	initFailed  bool
	root        generator
}

var (
	ErrNotImplemented    = errors.New("not implemented")
	ErrSyntaxInitialized = errors.New("syntax already initialized")
	ErrSyntaxInitFailed  = errors.New("syntax init failed")
	ErrNoDefinitions     = errors.New("syntax contains no definitions")
	ErrInvalidSyntax     = errors.New("invalid syntax")
)

func NewSyntax(o Options) *Syntax {
	if o.Trace == nil {
		o.Trace = NewTrace(TraceInfo)
	}

	return &Syntax{
		registry: newRegistry(),
		trace:    o.Trace,
	}
}

func (s *Syntax) ReadDefinition(r io.Reader) error {
	if s.initialized {
		return ErrSyntaxInitialized
	}

	if s.initFailed {
		return ErrSyntaxInitFailed
	}

	panic(ErrNotImplemented)
}

func (s *Syntax) register(d definition, ct CommitType) error {
	if s.initialized {
		return ErrSyntaxInitialized
	}

	if s.initFailed {
		return ErrSyntaxInitFailed
	}

	if err := s.registry.register(d); err != nil {
		return err
	}

	if ct&Root != 0 {
		return s.registry.setRoot(d.nodeName())
	}

	return nil
}

func (s *Syntax) AnyChar(name string, ct CommitType) error {
	return s.register(newChar(s.registry, name, true, false, nil, nil), ct)
}

func childName(name string, childIndex int) string {
	return fmt.Sprintf("%s:%d", name, childIndex)
}

func (s *Syntax) CharSequence(name string, ct CommitType, c []rune) error {
	var refs []string
	for i, ci := range c {
		ni := childName(name, i)
		refs = append(refs, ni)
		if err := s.register(
			newChar(s.registry, ni, false, false, []rune{ci}, nil),
			Alias,
		); err != nil {
			return err
		}
	}

	return s.Sequence(name, ct, refs...)
}

func (s *Syntax) Class(name string, ct CommitType, not bool, chars []rune, ranges [][]rune) error {
	return s.register(newChar(s.registry, name, false, not, chars, ranges), ct)
}

func (s *Syntax) Terminal(name string, ct CommitType, t ...Terminal) error {
	if len(t) == 0 {
		return ErrNoDefinitions
	}

	defs, err := terminalDefinitions(s.registry, name, t)
	if err != nil {
		return err
	}

	names := make([]string, len(defs))
	for i, d := range defs {
		if err := s.registry.register(d); err != nil {
			return err
		}

		names[i] = d.nodeName()
	}

	return s.register(newSequence(s.registry, name, ct, names), ct)
}

func (s *Syntax) Quantifier(name string, ct CommitType, item string, min, max int) error {
	return s.register(newQuantifier(s.registry, name, ct, item, min, max), ct)
}

func (s *Syntax) Sequence(name string, ct CommitType, items ...string) error {
	return s.register(newSequence(s.registry, name, ct, items), ct)
}

func (s *Syntax) Choice(name string, ct CommitType, items ...string) error {
	return s.register(newChoice(s.registry, name, ct, items), ct)
}

func (s *Syntax) Init() error {
	if s.initialized {
		return ErrSyntaxInitialized
	}

	if s.initFailed {
		return ErrSyntaxInitFailed
	}

	rootDef := s.registry.root
	if rootDef == nil {
		return ErrNoDefinitions
	}

	start := time.Now()
	root, ok, err := rootDef.generator(s.trace, "", nil)
	if err != nil {
		return err
	}

	log.Println("generator created", time.Since(start))

	if !ok {
		return ErrInvalidSyntax
	}

	// start = time.Now()
	// for {
	// 	var foundVoid bool
	// 	for id, g := range s.registry.generators {
	// 		g.finalize(s.trace)
	// 		if g.void() {
	// 			delete(s.registry.generators, id)
	// 			foundVoid = true
	// 		}
	// 	}

	// 	if !foundVoid {
	// 		break
	// 	}
	// }

	// log.Println(
	// 	"validation done",
	// 	time.Since(start),
	// 	len(s.registry.generators),
	// 	len(s.registry.definitions),
	// )

	if root.void() {
		return ErrInvalidSyntax
	}

	s.root = root
	s.initialized = true
	return nil
}

func (s *Syntax) Generate(w io.Writer) error {
	if !s.initialized {
		if err := s.Init(); err != nil {
			return err
		}
	}

	panic(ErrNotImplemented)
}

func (s *Syntax) Parse(r io.Reader) (*Node, error) {
	if !s.initialized {
		if err := s.Init(); err != nil {
			return nil, err
		}
	}

	c := newContext(bufio.NewReader(r))
	p := s.root.parser(s.trace, nil)
	return parse(p, c)
}