package mml

func syntax5() (*syntax, error) {
	s := newSyntax()

	var err error
	withErr := func(f func() error) {
		if err != nil {
			return
		}

		err = f()
	}

	withErr(func() error { return s.primitive("int", intToken) })
	withErr(func() error { return s.primitive("string", stringToken) })
	withErr(func() error { return s.optional("optional-int", "int") })
	withErr(func() error { return s.optional("int-repetition-optional", "int-repetition") })
	withErr(func() error { return s.repetition("int-repetition", "int") })
	withErr(func() error { return s.repetition("optional-int-repetition", "optional-int") })
	withErr(func() error { return s.sequence("single-int", "int") })
	withErr(func() error { return s.sequence("single-optional-int", "optional-int") })
	withErr(func() error { return s.sequence("multiple-ints", "int", "int", "int") })
	withErr(func() error { return s.sequence("sequence-with-optional-item", "optional-int", "string") })
	withErr(func() error { return s.choice("int-or-string", "int", "string") })
	withErr(func() error { return s.choice("int-or-sequence-with-optional", "int", "sequence-with-optional-item") })

	return s, err
}