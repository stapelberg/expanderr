(require 'go-mode)  ; we use a few gofmt-- functions.

(defcustom go-expanderr-command "expanderr"
  "The `expanderr' command; by default, $PATH is searched."
  :type 'string
  :group 'expanderr)

(defun go-expanderr ()
  "Expand the Call Expression before/under the cursor to check errors."
  (interactive)
  (let ((tmpfile (make-temp-file "expanderr" nil ".go"))
        (patchbuf (get-buffer-create "*Expanderr patch*"))
        (errbuf (if gofmt-show-errors (get-buffer-create "*Expanderr Errors*")))
        (coding-system-for-read 'utf-8)
        (coding-system-for-write 'utf-8)
        our-gofmt-args)

    (unwind-protect
        (save-restriction
          (widen)
          (if errbuf
              (with-current-buffer errbuf
                (setq buffer-read-only nil)
                (erase-buffer)))
          (with-current-buffer patchbuf
            (erase-buffer))

	  (save-buffer)
          (setq expanderr-command go-expanderr-command)
          (setq our-expanderr-args (list "-w" tmpfile
					 (concat
					  (file-truename buffer-file-name)
					  (format ":#%d" (position-bytes (point))))))
          (message "Calling expanderr: %s %s" expanderr-command our-expanderr-args)
          ;; We're using errbuf for the mixed stdout and stderr output. This
          ;; is not an issue because expanderr -w does not produce any stdout
          ;; output in case of success.
          (if (zerop (apply #'call-process expanderr-command nil errbuf nil our-expanderr-args))
              (progn
                (if (zerop (call-process-region (point-min) (point-max) "diff" nil patchbuf nil "-n" "-" tmpfile))
                    (message "Buffer is already expanded")
                  (go--apply-rcs-patch patchbuf)
                  (message "Applied expanderr"))
                (if errbuf (gofmt--kill-error-buffer errbuf)))
            (message "Could not apply expanderr")
            (if errbuf (gofmt--process-errors (buffer-file-name) tmpfile errbuf))))

      (kill-buffer patchbuf)
      (delete-file tmpfile))))

(add-hook 'go-mode-hook (lambda ()
			  (local-set-key (kbd "C-c C-e") #'go-expanderr)))

(provide 'go-expanderr)
