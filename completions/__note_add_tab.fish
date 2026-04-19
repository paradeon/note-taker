function __note_add_tab
    set -l tok (commandline -ct)
    if string match -q ',,*' -- $tok
        set -l prev (commandline -opc)
        if contains note $prev; and contains add $prev
            # Resolve notes file from -f/--file flag or NOTE_FILE/default
            set -l file ""
            for i in (seq 1 (count $prev))
                switch $prev[$i]
                    case '-f' '--file'
                        set -l next (math $i + 1)
                        if test $next -le (count $prev)
                            set file $prev[$next]
                        end
                end
            end
            if test -z "$file"
                if set -q NOTE_FILE
                    set file $NOTE_FILE
                else
                    set file "$HOME/notes/quick-notes.md"
                end
            end

            set -l matches (note completions --file $file $tok 2>/dev/null)
            if test (count $matches) -eq 1
                # Single match: insert with trailing comma, no space, ready to chain
                commandline -t -- $matches[1],
                commandline -f repaint
                return
            end
        end
    end
    commandline -f complete
end
