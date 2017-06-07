ws:alias  = " " | "\t" | "\n" | "\b" | "\f" | "\r" | "\v";
wsc:alias = ws | comment;

block-comment:alias   = "/*" ("*" [^/] | [^*])* "*/";
line-comment:alias    = "//" [^\n]*;
comment-segment:alias = line-comment | block-comment;
ws-no-nl:alias        = " " | "\t" | "\b" | "\f" | "\r" | "\v";
comment               = comment-segment (ws-no-nl* "\n"? ws-no-nl* comment-segment)*;

any-char = "."; // equivalent to [^]

// caution: newline is accepted
class-not  = "^";
class-char = [^\\\[\]\^\-] | "\\" .;
char-range = class-char "-" class-char;
char-class = "[" class-not? (class-char | char-range)* "]";

// caution: newline is accepted
sequence-char = [^\\\"] | "\\" .;
char-sequence = "\"" sequence-char* "\"";

// TODO: this can be mixed up with sequence. Is it fine? fix this, see mml symbol
terminal:alias = any-char | char-class | char-sequence;

symbol = [^\\ \n\t\b\f\r\v/.\[\]\"{}\^+*?|():=;]+;

group:alias = "(" wsc* expression wsc* ")";

count            = [0-9]+;
count-quantifier = "{" wsc* count wsc* "}";
range-quantifier = "{" wsc* count wsc* "," wsc* count wsc* "}";
one-or-more      = "+";
zero-or-more     = "*";
zero-or-one      = "?";
quantity:alias   = count-quantifier
                 | range-quantifier
                 | one-or-more
                 | zero-or-more
                 | zero-or-one;

quantifier = (terminal | symbol | group) wsc* quantity;

item:alias = terminal | symbol | group | quantifier;
sequence   = item (wsc* item)+;

element:alias = terminal | symbol | group | quantifier | sequence;
choice        = element (wsc* "|" wsc* element)+;

expression:alias = terminal
                 | symbol
                 | group
                 | quantifier
                 | sequence
                 | choice;

alias      = "alias";
doc        = "doc";
root       = "root";
flag:alias = alias | doc | root;
definition = symbol (":" flag)* wsc* "=" wsc* expression;

definitions:alias = definition (wsc* ";" (wsc | ";")* definition)*;
document:root     = (wsc | ";")* definitions? (wsc | ";")*;
