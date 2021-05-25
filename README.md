# Interactive diff

A simple tool to compare Frontend and DCR interactives as we migrate them.

To run, simply grab the latest release and execute it:

    $ PORT=8080 ./interactive-diff

To compare against local DCR, use the `--local` flag:

    $ PORT=8080 ./interactive-diff --local

Note, this will only work if you are running DCR on http://localhost:3030
because.

## Development

Use Gin for watch/reload:

    $ go get github.com/codegangsta/gin
    $ gin -all run main.go
