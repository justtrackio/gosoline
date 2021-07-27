## Testing with streams

So now we have written our nice stream-based code.
What is next?
We could simply release our untested code to the world.
Maybe we even did test it once.
It would however be much better if we can write an automated, repeatable test, which ensures us our code is doing what it is supposed to be doing.
Thus, we go back, read how these inputs worked again, and write a consumer which reads the messages we published and verifies them.
Hmm, that sounds like a lot of unnecessary work.
There has to be an easier way...

