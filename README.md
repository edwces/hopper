# Web crawler in Go

This is basic web crawler/scraper implementation.

## Missing Features/Bugs that need fixes

- [ ] Workers are busy with delayed items. Because there is not any
prioritization of **NEW** request with no delay. The delay proccess is blocking
each worker while instead it should be each transfered in it's own goroutine and then send to
priority queue
- [ ] URLQueue can only be stored in memory which in longer crawling end up unsufficient
- [ ] Better error and configuration handling would be nice
- [ ] Each request should have it's own goroutine instead of having goroutines of loops
- [ ] Some more test but only after this entire rewrite
