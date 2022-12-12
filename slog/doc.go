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
as a string and passes it to the [log] package.

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

Both [TextHandler] and [JSONHandler] can be configured with a [HandlerOptions].
There are options for setting the minimum level (see Levels, below),
displaying the source file and line of the log call, and
modifying attributes before they are logged.

Setting a logger as the default with

    slog.SetDefault(logger)

will cause the top-level functions like [Info] to use it.
[SetDefault] also updates the default logger used by the [log] package,
so that existing applications that use [log.Printf] and related functions
will send log records to the logger's handler without needing to be rewritten.


# Attrs and Values

An [Attr] is a key-value pair. The Logger output methods accept Attrs as well as
alternating keys and values. The statement

    slog.Info("hello", slog.Int("count", 3))

behaves the same as

    slog.Info("hello", "count", 3)

There are convenience constructors for [Attr] such as [Int], [String], and [Bool]
for common types, as well as the function [Any] for constructing Attrs of any
type.

The value part of an Attr is a type called [Value].
Like an [any], a Value can hold any Go value,
but it can represent typical values, including all numbers and strings,
without an allocation.

For the most efficient log output, use [Logger.LogAttrs].
It is similar to [Logger.Log] but accepts only Attrs, not alternating
keys and values; this allows it, too, to avoid allocation.

The call

    logger.LogAttrs(slog.LevelInfo, "hello", slog.Int("count", 3))

is the most efficient way to achieve the same output as

    slog.Info("hello", "count", 3)

Some attributes are common to many log calls.
For example, you may wish to include the URL or trace identifier of a server request
with all log events arising from the request.
Rather than repeat the attribute with every log call, you can use [Logger.With]
to construct a new Logger containing the attributes:

    logger2 := logger.With("url", r.URL)

The arguments to With are the same key-value pairs used in [Logger.Info].
The result is a new Logger with the same handler as the original, but additional
attributes that will appear in the output of every call.



# Levels

A [Level] is an integer representing the importance or severity of a log event.
The higher the level, the more severe the event.
This package defines constants for the most common levels,
but any int can be used as a level.

In an application, you may wish to log messages only at a certain level or greater.
One common configuration is to log messages at Info or higher levels,
suppressing debug logging until it is needed.
The built-in handlers can be configured with the minimum level to output by
setting [HandlerOptions.Level].
The program's `main` function typically does this.
The default value is LevelInfo.

Setting the [HandlerOptions.Level] field to a [Level] value
fixes the handler's minimum level throughout its lifetime.
Setting it to a [LevelVar] allows the level to be varied dynamically.
A LevelVar holds a Level and is safe to read or write from multiple
goroutines.
To vary the level dynamically for an entire program, first initialize
a global LevelVar:

    var programLevel = new(slog.LevelVar) // Info by default

Then use the LevelVar to construct a handler, and make it the default:

    h := slog.HandlerOptions{Level: programLevel}.NewJSONHandler(os.Stderr)
    slog.SetDefault(slog.New(h))

Now the program can change its logging level with a single statement:

    programLevel.Set(slog.LevelDebug)


# Groups

Attributes can be collected into groups.
A group has a name that is used to qualify the names of its attributes.
How this qualification is displayed depends on the handler.
[TextHandler] separates the group and attribute names with a dot.
[JSONHandler] treats each group as a separate JSON object, with the group name as the key.

Use [Group] to create a Group Attr from a name and a list of Attrs:

    slog.Group("request",
        slog.String("method", r.Method),
        slog.Any("url", r.URL))

TextHandler would display this group as

    request.method=GET request.url=http://example.com

JSONHandler would display it as

    "request":{"method":"GET","url":"http://example.com"}

Use [Logger.WithGroup] to qualify all of a Logger's output
with a group name. Calling WithGroup on a Logger results in a
new Logger with the same Handler as the original, but with all
its attributes qualified by the group name.

This can help prevent duplicate attribute keys in large systems,
where subsystems might use the same keys.
Pass each subsystem a different Logger with its own group name so that
potential duplicates are qualified:

    logger := slog.Default().With("id", systemID)
    parserLogger := logger.WithGroup("parser")
    parseInput(input, parserLogger)

When parseInput logs with parserLogger, its keys will be qualified with "parser",
so even if it uses the common key "id", the log line will have distinct keys.

# Contexts

# Advanced topics



## Customizing a type's logging behavior

TODO: discuss LogValuer

## Wrapping output methods

TODO: discuss LogDepth, LogAttrDepth

## Interoperating with other logging packages

TODO: discuss NewRecord, Record.AddAttrs

## Performance considerations

TODO: mention to avoid unnecessary work (like calling URL.String()) in log args in case
the log statement is disabled.

## Writing a handler

*/
