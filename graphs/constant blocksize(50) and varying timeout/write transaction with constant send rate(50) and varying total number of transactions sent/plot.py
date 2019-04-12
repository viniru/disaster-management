import numpy as np
import matplotlib.pyplot as plt
import pandas as pd

data=[]
x=[]
y=[]
for i in range(0,3):
    source = "graph" + str(i+1) + ".txt"
    data.append(pd.read_csv(source))
    x.append(data[i].iloc[:,0:1].values)
    y.append(data[i].iloc[:,1:2].values)



plt.plot(x[0],y[0],color="red",label="1500ms")
plt.plot(x[1],y[1],color="blue",label="3000ms")
plt.plot(x[2],y[2],color="green",label="4000ms")
plt.title("Constant block size and send rate with varying timeout and (# of transactions vs throughput)")
plt.xlabel("# of transactions")
plt.ylabel("throughput")
plt.legend()
plt.savefig("graph")
plt.show()
