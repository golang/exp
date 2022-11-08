// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slog

/*
Package slog provides structured logging,
in which log records include a message,
a severity level, and various other attributes
expressed as key-value pairs.

It defines a type, [Logger],
which provides several methods (such as [Logger.Info] and [Logger.Error])
for reporting events of interest.

Each Logger is associated with a [Handler].
A Logger output method creates a [Record] from the method arguments
and passes it to the Handler, which decides how to handle it.
There is a default Logger accessible through top-level functions
(such as [Info] and [Error]) that call the corresponding Logger methods.

A log record consists of a time, a level, a message, and a set of key-value
pairs, where the keys are strings and the values may be of any type.
As an example,

    slog.Info("hello", "count", 3)

creates a record containing the time of the call,
a level of Info, the message "hello", and a single
pair with key "count" and value 3.

The [Info] top-level function calls the [Logger.Info] method on the default Logger.
In addition to [Logger.Info], there are methods for Debug, Warn and Error levels.
Besides these convenience methods for common levels,
there is also a [Logger.Log] method which takes the level as an argument.
Each of these methods has a corresponding top-level function that uses the
default logger.

The default handler formats the log record's message, time, level, and attributes
as a string and passes it to the [log] package."

    2022/11/08 15:28:26 INFO hello count=3

For more control over the output format, create a logger with a different handler.
This statement uses [New] to create a new logger with a TextHandler
that writes structured records in text form to standard error:

    logger := slog.New(slog.NewTextHandler(os.Stderr))

[TextHandler] output is a sequence of key=value pairs, easily and unambiguously
parsed by machine. This statement:

    logger.Info("hello", "count", 3)

produces this output:

    time=2022-11-08T15:28:26.000-05:00 level=INFO msg=hello count=3

The package also provides [JSONHandler], whose output is line-delimited JSON:

    logger := slog.New(slog.NewJSONHandler(os.Stdout))
    logger.Info("hello", "count", 3)

produces this output:

    {"time":"2022-11-08T15:28:26.000000000-05:00","level":"INFO","msg":"hello","count":3}

Setting a logger as the default with

    slog.SetDefault(logger)

will cause the top-level functions like [Info] to use it.
[SetDefault] also updates the default logger used by the [log] package,
so that existing applications that use [log.Printf] and related functions
will send log records to the logger's handler without needing to be rewritten.


# Attrs and Values

# Levels

# Configuring the built-in handlers

TODO: cover HandlerOptions, Leveler, LevelVar

# Groups

# Contexts

# Advanced topics

## Customizing a type's logging behavior

TODO: discuss LogValuer

## Wrapping output methods

TODO: discuss LogDepth, LogAttrDepth

## Interoperating with other logging packabes

TODO: discuss NewRecord, Record.AddAttrs

## Writing a handler

*/
