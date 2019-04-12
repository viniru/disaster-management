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



plt.plot(x[0],y[0],color="red",label="10 Transactions/Block")
plt.plot(x[1],y[1],color="blue",label="20 Transactions/Block")
plt.plot(x[2],y[2],color="green",label="30 Transactions/Block")
plt.title("Constant Total transactions and timeout with varying block size(# of transactions vs throughput)")
plt.xlabel("Send rate")
plt.ylabel("Throughput")
plt.legend()
plt.savefig("graph")
plt.show()
