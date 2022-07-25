# What is that?

Stress testing Docker filesharing.

# How to run

Docker must be installed in order to compile the code and run the test.

```bash
make
```

Tweak the number of tests that run concurrently:

```bash
COUNT=10 make run
```