package main

/*
```
This block opens a fence at the left margin and never closes it. A markdown
section scan run over this file refuses it as an unterminated block, which is
why the body redaction is scoped to markdown and runs on nothing else.
*/

func fenced() {}
