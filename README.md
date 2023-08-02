# pow-test-task

## task

Design and implement “Word of Wisdom” tcp server.

• TCP server should be protected from DDOS attacks with the Prof of Work
(https://en.wikipedia.org/wiki/Proof_of_work), the challenge-response protocol should
be used.

• The choice of the POW algorithm should be explained.

• After Prof Of Work verification, server should send one of the quotes from “word of
wisdom” book or any other collection of the quotes.

• Docker file should be provided both for the server and for the client that solves the
POW challenge.

## choice of the POW algorithm

I chose the [Hashcash](https://en.wikipedia.org/wiki/Hashcash) scheme with SHA-1 hash.

This algorithm isn't ASIC-resistant and even having a good GPU makes participants really unequal.

I'm sure a better (fair) PoW algorithm exists, but I haven't found it.

A pro of Hashcash is that its PoW can be easily made harder or easier changing the number of zeros.

Another pro of SHA-1 is its uniform distribution. It makes PoW almost equal amount of computations for different participants.

## local installation

`docker-compose up`

## how to test locally

Open localhost in your browser.

To tune the client and server, edit `.env`.

## implementation

The actual PoW logic is in `client/pow/pow.go` and `server/check/check.go`.