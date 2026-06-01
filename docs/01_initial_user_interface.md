# Initial user interface

## overview

The initial user interface will be intentionally limited, so we can experiment.

On the left we show all the services, with their logs.
There is a cursor so the user can navigate with up/down arrow (or k/j, vi-style). To the right of it is the main pane, where we show the content of log(s).

## left pane

Here is an example left pane:

>○  api              |
   - log.txt         >
 ● workers ▂         |
   ▷ worker_1 ▇      |
     - stdout.txt ▇  |
     - stderr.txt ▇  |
   ▷ worker_2 ▇      |
     - stdout.txt ▇  |
     - stderr.txt ▇  |

"api" and "workers" are services: they come from the configuration file. Next to each service name is a **liveness** indicator from the `health` block in config: **●** when probes say the service is up, **○** when probes are configured but it looks down, **-** when no health probes are configured.

worker_1 and worker_2 are folders that the configuration for "workers" said to watch for log files.

log.txt, stdout.txt and stderr.txt are individual log files.
Next to them is a vertical bar, the "log activity indicator".
The bar fills 100% (this is just unicode so it has 8 degrees of fullness to choose from) when the log file changes, and then slowly empties over time. This gives a visual indication of which files are active.

The ">" on the left indicates the user's cursor. Right now the cursor is on "api".

The "|" on the right is the edge of the left pane. It's broken by a ">" symbol on the specific log that is being shown in the main pane right now. At this moment, the main pane shows the contents of the log.txt file of the "api" service.

When the user presses up/down it moves the cursor. The row under the cursor ("selected row") is also displayed with a background highlight color so it stands out.

If the user presses ENTER on a line then the currently selected line (probably a log, but could also be a service) becomes shown: there is only one ">" symbol on the right and it's on the row for that line.

If the user presses "+" then the currently selected line is shown *in addition* to the previously shown one(s). Then there will be two ">" symbols on the right. If they press "-" then the currently selected line will no longer be shown: it will be removed from the main panel and there won't be a ">" next to it anymore (if that was the last one then the main panel will be empty)

## main pane

### Showing log(s)

What's actually shown in the main panel is the content of the log, automatically updated if the file content changes.

New implementation: showing multiple logs:

look for the 'timestamp' line variable in each log and show in increasing order of those.
If one or more logs do not have a timestamp line variable then we refuse to add them to the set being shown when the user presses '+', and we show "can only combine logs with parsed timestamps" in the status bar until the next keypress.


### Showing a service

When showing a service instead of a log, the main pane shows information about the service (also subtly explains what the symbols mean)

example:

```
● workers ▂ 

● port 4500 bound to PID 5432
  running since: before 2025-6-1 08:00
▂ 1mim activity:  0.5 lines/s
▂ 10min activity: 0.2 lines/s

  [ KILL ]

2025-06-01 15:03:00.311 workers PID 5432 running on port 4500
```

This example shows information we're tracking for all processes:
- running/stopped
- last start time
- lines written in the last 10min

The liveness information line will depend on how the configuration file is configured:

port configured and visible:
```
● port 4500 bound to PID 5432
```

proc name configured and visible:
```
● process name "loggen" bound to PID 5432
```

both port and proc name configured and visible:
```
● port 4500 bound to process name "loggen", PID 5432
```

configured but not visible: one of
```
○ port 4500, nothing bound
○ process name "loggen" not found
○ port 4500, bound but not to process name "loggen"
○ port 4500, not bound. Process name "loggen" exists.
```

Not configured:
```
- no liveness detection configured for this service
```

The process information in the main pane updates in real time, including the activity indicators. For now, they both show the exact same thing.

Underneath this are buttons. They will be configurable through the config file but for now we will only show a "KILL" button if we know the PID for this service. Pressing "Kill" will send a SIGTERM signal to that process.

Underneath the buttons is a "synthetic log" that we keep for all processes. It's a list of relevant events:

- running: process is running as our app starts
- starts: process goes from not running to running
- SIGTERM sent
- stops: process goes from running to not running

For now we won't have a way to actually press the buttons, that's for later.

### showing both a service and one or more logs (future implementation)

Then we mix the logs and the service's synthetic log, same rules as mixing multiple logs.

