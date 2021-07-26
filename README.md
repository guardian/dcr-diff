# DCR diff

A simple tool to compare Frontend and DCR interactives as we migrate them.

The tool reads from a Google doc for the list of source URLs and then updates
the doc based on user feedback. Any Google spreadsheet that you have write
access to can be used but the first three columns must be:

    URL, Status, Comment

(You can add additional columns on the right though with any extra data you
want.)

The tool will show subsequent rows where the URL is populated but the Status is
empty.

To run, simply grab the latest release and execute it:

    $ PORT=8080 ./dcr-diff --spreadsheet [url] --sheetID [sheet-id]

Where url is like: https://docs.google.com/spreadsheets/d/[some ID]/edit.

Quote the sheet ID if more than a single word.

## Development

Use Gin for watch/reload:

    $ go get github.com/codegangsta/gin
    $ SPREADSHEET=[url] SHEETID=[id] gin -all run .

Note, required to use env vars as gin doesn't support passing flags directly.
