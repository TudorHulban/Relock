# Relock
Redis locking

## Theory
Message broker MB sends event with ID id12345 to two subscriber consumers, C1 and C2.   
On sucessful processing the consumer would send an acknowledgment to MB. Based on this acknowledgement MB will not resend again the event.  
For locking Redis nodes are used in muber of N = 5.

### Locking - case 1
MB resends the event after TResend = 10 seconds.  
Processing time for the event for both C1 and C2 is TProcess = 6 seconds.  
Redis release of lock time is TRelease = 7 seconds.
Retry for gaining lock for C1 and C2 is TRetryInterval = 1 second.  
Number of retries is NoRetries = 3.   

#### Locking - sequence of events 1
1. At time NOW an event is sent by MB to C1 and C2.
2. C1 and C2 receive the event and go for the lock. 
3. C1 gets the lock and starts processing the event.
4. at NOW + TRetryInterval C2 tries again to get the lock
5. at NOW + TRetryInterval C2 tries again to get the lock
6. at NOW + TRetryInterval C2 tries again to get the lock
7. at NOW + TProcess C1 finishes processing the event, updates the persistance and MB and releases the lock.

#### Locking - sequence of events 2
1. At time NOW an event is sent by MB to C1 and C2.
2. C1 and C2 receive the event and go for the lock. 
3. C1 gets the lock and starts processing the event.
4. at NOW + TRetryInterval C2 tries again to get the lock
5. at NOW + TRetryInterval C2 tries again to get the lock
6. at NOW + TRetryInterval C2 tries again to get the lock
7. at NOW + 5 seconds C1 crashes
8. at NOW + TRelease the Redis nodes release the lock.
9. at NOW + TResend MB resends event with id12345
10. C2 sucesfully locks the processing for event with id12345
11. at NOW + TResend + TProcess C2 finishes processing the event, updates the persistance and MB and releases the lock.