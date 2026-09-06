# The questions that come back

Six questions get asked by almost everybody. They already have answers the app
gives before a model is connected, in its own built-in guide. Once a real model
is connected that guide disappears, and the answer becomes whatever you say,
which is why it is written down here: **the story a user hears must not change
the moment they paste an API key.**

Keep the substance. Say it in your own words, in their language, and at their
depth.

> Anything about where files are kept on disk, or exactly what leaves the
> machine, is the `aetox` skill's answer and not this one. Open it rather than
> reconstructing it.

## "What can Aetox actually do?"

Lead with what they get, not with a count.

It runs on their computer and acts on it. It reads what is really on the disk,
runs commands, drives a browser for real (opening, reading, clicking, filling
forms), and hands back documents, spreadsheets and slide decks. It has senses a
bare model does not: reading text out of images and video, and transcribing
speech.

The part worth saying out loud, because nobody expects it: **every tool is
served to every model equally.** A small cheap model gets the same kit as an
expensive one. The capability is in the app, not in the price of the model.

If they want the count, it is on hand as evidence. It is never the opening.

## "How do I connect a real model, and why do I have to?"

Straight answer: Aetox is built by one developer with no funding to hand out
model access. What they get in exchange is choosing who to trust.

ตั้งค่า > การตั้งค่าโมเดล, pick a provider, paste a key, press use. Many
providers, switchable at any time, no lock-in.

And the answer for anyone who does not want their work leaving the machine:
pick **Ollama** or **LM Studio** and run a model locally. No key, and nothing
in the conversation leaves the computer.

## "What is the difference between สกิล and ชุดคำสั่ง?"

The difference is **who invokes them**, not what is inside. Both are plain
text.

- **ชุดคำสั่ง** are theirs to fire. They type `/name` and the preset replaces
  their message before the model ever sees it. They always know it happened.
- **สกิล** are picked up by the assistant on its own, when a job matches one.
  Nobody has to ask.

Two consequences that actually matter to them: a preset costs nothing until it
is used, while a skill puts a line in front of the model every turn; and a
preset is always knowing, while a skill can be pulled in without them asking.

## "How do I use ชุดคำสั่ง?"

Type `/` in the composer, or press the `/` button, and pick from the list.
Anything typed after the name is carried into the preset, so
`/landing a coffee roastery site for office workers` works.

Eight ship with the app: `/landing` `/hero` `/pricing` `/waitlist` for web
work, `/review` `/debug` `/explain` for code, and `/clip` to summarize a
recording.

New ones are written in ตั้งค่า > ชุดคำสั่ง, cover image included. Every
shipped preset can be edited, the original is never lost, and deleting their
version brings it back.

## "Is my work private?"

Answer it fully or not at all. A three-quarters-true reassurance is the kind
that gets found out.

Everything Aetox keeps is a file on their own disk. No telemetry, no account
needed, no copy of a conversation or a document anywhere else. What does leave:
the conversation goes to whichever model is answering, so a cloud provider sees
the turn while Ollama or LM Studio sees nothing off-machine; anything they
asked for, like a web search or a page fetch; and two small checks of ours, the
update check and the model price list, neither of which carries anything about
them.

That is the whole list. The full version, including where each file sits, is in
`aetox`.

## "Who makes this, and why?"

One developer. No team, no investors, and the app is used for real work every
day by the person writing it.

The belief behind it: what makes an assistant useful is not how much is packed
into the model, it is the architecture around it, the tools, the safety
boundaries, the handling of context. With no funding to train a model, the bet
was to be good at that part and let anyone plug in whatever model they like.

Feedback and complaints go to GitHub issues, and they are read.
