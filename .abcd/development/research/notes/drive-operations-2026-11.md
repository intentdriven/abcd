# Operating an unattended drive, 11 to 14 November 2026

## Summary

`abcd drain --drive` ran unattended for three days on one machine. It landed work, but keeping it running took a second session watching it, and bringing it back after it stalled took a person six attempts over six hours. None of the steps were ones a product thinker or a technical facilitator could be expected to know. This note records what happened, in order, and what each event shows. It is the input to the lab on drive operations and to the intent that follows it.

## What the drive did

- First run: about 52 hours. It opened 22 pull requests, of which 20 merged, and handed back 79 items: 48 at its own judgement before a lane opened, 31 from lanes, 20 of those as stuck.
- For the last 23 of those hours it did no new work and did not say so in any way a person would notice.
- Second run, after a rebuild: six hours alive, no lane opened.
- Third run, after the person closed a terminal window: working within a minute.

## Events

### 1. A stage could not take the reservation its own brief told it to take

A walled stage holds no credential for a reservation. Its brief told it to widen the lane's hold when its work reached one more file. Each attempt failed, the launch was repeated three times, and the lane was handed back as stuck. Eleven of the twenty stuck lanes fit this. The loop now takes the reservation itself. The drive kept running the older build, so it kept failing the same way until a person rebuilt it.

What it shows: a fix on the default branch does not reach a drive that is running. Nothing tells the person that the drive they started is now behind a fix it needs.

### 2. A reviewer's verdict was refused for naming the wrong thing

Eight lanes ended as stuck although every reviewer had passed the work: the verdict line named the issue's digits where the validator expected the commit. The fix was on the default branch before the drive started its second day, and the drive did not have it.

What it shows: the same as event 1. Also, a lane whose work is finished and passed is thrown away whole when its last step is refused; nothing keeps the passed work for a later build to land.

### 3. Pull requests left the merge queue and stayed out

Four of the drive's pull requests were removed from the merge queue by a test that fails now and then on the forge's runners, or by a runner that killed a process. The drive read each removal correctly and printed what a person should do: re-run the failed check, then arm the pull request again with a command it spelled out, then step the run. It did none of that itself, and printed the same paragraph every thirty seconds. A second session re-ran the checks and re-armed each one by hand.

A fifth pull request had workflow runs the forge created and never started. The forge refuses to cancel or re-run them. It is still open.

What it shows: the drive can tell a removal that follows a real failure of the change from one that does not, and already holds everything it needs to re-arm. Leaving that to a person makes every flaky check a manual step.

### 4. The drive's checkout fell behind, and one missing record stopped everything

The drive runs from a checkout it never updates. After two days that checkout was 692 commits behind the default branch. The drive judged an issue it read from origin, built its lane, and at the landing looked the issue's record up in its own checkout to find the path to reserve. The record was not there. It waited fifteen minutes and looked again, 47 times, and opened no other lane in between.

What it shows: one lane that cannot proceed holds the whole drive. The wait had no bound and no escalation. The log line said what was missing and never said that nothing had moved for 23 hours.

### 5. Nobody could tell whether it was working

The person's question on return was "is the drain working?". Answering it took reading the log for the last line that was not a wait, comparing times, and counting. The drive prints its own state as a stream of similar lines. There is no one line that says: working, waiting on a limit, or stalled since a given time for a given reason.

### 6. Bringing it back took a rebuild by hand

To restart the drive on a build with the fixes, a session had to:

1. fast-forward the checkout, which the drive's own uncommitted records blocked (28 draft intents and 44 back-links it had written and never committed);
2. set one of those files aside, fast-forward, and put it back;
3. build the binary by hand to the place the drive runs from, keeping the old one;
4. edit the machine's configuration, because the bundled review pair now names a model the provider block did not list.

Each step was found by reading code or by a refusal.

### 7. A stop by signal left a mark only a take-over lifts

The person stopped the first drive with an interrupt. The next start printed thirteen lines of plan and then one paragraph: a stop set by a signal holds, nothing further starts, and the mark stays until the run is taken over. The paragraph named two commands and the difference between them in terms of a stop's mark and a schema version.

### 8. The take-over made things worse

The person ran the take-over the message named, in the same terminal, and started the drive. The take-over had recorded the terminal's login process as the run's driver. The drive, started from that terminal, read the run as driven by another live session and waited on it. That process lives as long as the terminal does. The drive opened no lane for six hours.

### 9. The advice given to the person was wrong, twice

The watching session told the person to take the run over from a second terminal and close that one. The take-over was refused: the recorded driver was still alive in the first terminal. An interrupt to the drive could not set its stop either, for the same reason. What worked was closing the first terminal, which nothing had said.

What events 7 to 9 show: stopping and starting a drive depends on which process a record names, on whether a mark stands, on the schema version of a run file and on which terminal a command is typed in. A person cannot be expected to hold that model, and a session that had read the code got it wrong.

### 10. Other things a person had to know

- The drive holds a reservation over whatever its lane touches, sometimes a whole directory, and another account's work waits on it with no way to ask.
- Lanes handed back as stuck keep a worktree, a branch and a mark that stops any drain taking the issue again. Clearing twenty of them is twenty commands a person types.
- A newer default branch can carry a lint rule name an older binary stops on. It did not happen here; it was one message away from happening twice.
- The drive shares its subscription's budget with every other session of the account, and says nothing about how much of the window it has used.

## What a person had to do, counted

Over the three days the drive needed, from a person or a session acting for one: four re-runs of forge checks, four re-arms, one rebuild in four steps, one configuration edit, one take-over, one closed terminal, and the reading of roughly forty thousand log lines to learn its state. It asked for none of these in a way that reached anyone.

## Questions for the lab

1. After an interrupt, a kill, a closed terminal or a reboot, does one start command bring the drive back with nothing else typed?
2. Can the drive take up its own earlier run whatever its recorded driver and schema version, when that driver is gone or is the terminal the drive itself runs in?
3. Can the drive bring its own checkout and build up to the default branch between lanes, and what must be true for that to be safe?
4. When a lane cannot proceed, can the drive set it aside and open the next one, and after how long?
5. Can the drive re-run a check and re-arm a pull request the forge removed for a reason that is not the change's own, and how does it tell?
6. What is the one line that answers "is it working?", and where does a person see it without reading the log?
