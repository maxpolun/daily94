;
; this is a sample list file for this implementation of list in go
;

(print "print works")
(cond ((> 2 1) (print "cond works")))
(print (quote quote-works))
((lambda (s) (print s)) "lambda works")
