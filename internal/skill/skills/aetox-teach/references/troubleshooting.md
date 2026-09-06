# When it is not working

The state this file is for: the user is about to conclude the app is broken.
Usually it is not, and usually the real cause is one of six things.

Two rules for all of them. **Check before saying**, because a confident wrong
diagnosis costs more than a question. And **name the page, not the file**: a
user cannot act on a path, they can act on ตั้งค่า > something.

## Nothing happens when they send a message

Almost always no model connected, and it is worth checking first every time
because it is the one that makes every other symptom look mysterious.

ตั้งค่า > การตั้งค่าโมเดล, pick a provider, paste a key, press use. If they
would rather nothing left the machine, Ollama or LM Studio, no key needed.

## The key was rejected

Three causes, in the order they actually happen: the key was pasted with a
space or a newline in it, the key belongs to a different provider than the one
selected, or the account has no credit. Say which one to check rather than
suggesting they try again.

## A folder is refused

Some folders are refused in every mode, whatever the user added, and this is
not a bug or a setting. Credential stores are the obvious ones. Aetox's own
skills folder is also refused to the file tools on purpose; skills are read
through their own door instead, and the refusal message says so.

What to do: name the folder, say plainly that it is refused rather than that
something failed, and ask for a different one. Do not try a second spelling of
the same path.

## A tool that should be there is missing

Two different causes with two different answers:

- **An account is not connected**, or is not placed on this desk. A missing
  GitHub is not a failure, it means this desk does not carry it.
  ตั้งค่า > การเชื่อมต่อ.
- **A server was never added.** ตั้งค่า > MCP servers. You cannot add one
  yourself, so point at the page and carry on with the part that does not need
  it.

Then do the rest of the job. Refusing the whole thing over one missing piece is
the wrong answer.

## It answers, but does nothing

Check **โหมดทำงาน** before anything else. In **วางแผน** nothing gets written or
run, by design, and in **คู่คิด** there are no tools at all. Both are the user's
own choice, and both look exactly like a broken assistant if nobody says so.

Tell them which mode is on and that switching back is one press. Do not offer
to do it anyway.

## A local model is very slow, or gives up

Running the model on their own machine is a real trade: nothing leaves the
computer, and the speed is whatever the hardware gives. The usual causes are a
model larger than the graphics memory available, or two local runtimes loaded
at once and sharing it.

Say the trade honestly. A hosted provider for the heavy jobs and a local model
for the private ones is a normal way to work, not a defeat.

## When it really is a bug

Say so plainly. Do not invent a workaround that quietly produces a worse
result, and do not blame the user's setup for something that is ours. The
about page carries the recent internal log, and issues go to GitHub.
