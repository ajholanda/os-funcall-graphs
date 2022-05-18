LATEX := xelatex

main.pdf: main.aux $(maindeps)
	$(LATEX) main

main.aux: main.tex
	$(LATEX) main

TRASH += *~ main.aux main.log main.out main.pdf

clean:
	$(RM) $(TRASH)

.PHONY: clean
