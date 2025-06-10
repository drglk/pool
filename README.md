# Worker pool
## Descritrion

This repository implements a simple worker pool, in which the number of workers can be changed dynamically.

## Usage

To create a new pool, you need to call the New() function. It takes a string channel and a function to be processed.

Next, you need to start the pool using the Run() function. It takes the number of workers that should be created.
It should be noted that the pool cannot work without workers. Therefore, as a simple solution, it was decided that the Run() command creates at least one worker.

In the future you can change amount of workers by functions AddWorker() and RemoveWorker().

Also, at any time you can find out the current number of workers by function WorkersCount().

