;; this file provides backwards compatibilty with previously documented way of
;; loading this libray in the readme:
;; (load "~/go/src/github.com/stapelberg/expanderr/expanderr.el")
(load-file (expand-file-name "lisp/go-expanderr.el" (file-name-directory load-file-name)))
