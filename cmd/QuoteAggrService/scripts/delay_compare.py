import matplotlib.pyplot as plt
import pandas as pd
import numpy as np
import matplotlib.dates as mdates

def read_data(file='ws_client.log'):
    res = []
    with open(file, 'r') as f:
        lines = f.readlines()
        for line in lines:
            line = line.strip()
            t , left = line.split('[')[1].split(']')
            # print(t)
            t = t.split(',')[0]
            fs = left.split('dealy:')
            if len(fs) == 2:
                # print(fs[1].strip())
                res.append( [ t,float(fs[1].strip())] )
    return res

# aa = read_data('./log/ws_client.log.2024-09-03')
aa = read_data('./log/ws_client.log')
df1 = pd.DataFrame(aa, columns=['time', 'delay'])
df1.index = pd.to_datetime(df1['time'],format='%Y-%m-%d %H:%M:%S')

df1 = df1[df1['delay'] < 2000 ]

bb = read_data('./log/ws_client_zx.log')
df2 = pd.DataFrame(bb, columns=['time', 'delay'])
# df2.set_index('time', inplace=True)
# df2.index = pd.to_datetime(df2['time'],format='%Y-%m-%d %H:%M:%S') +  pd.Timedelta(hours=8)
df2.index = pd.to_datetime(df2['time'],format='%Y-%m-%d %H:%M:%S')
df2 = df2[df2['delay'] < 5000 ]

# 移动平均  15个数据
# df1['delay'] = df1['delay'].rolling(window=5).mean()
# df2['delay'] = df2['delay'].rolling(window=5).mean()

df1['delay'] = df1['delay'].rolling(window=15).min()
df2['delay'] = df2['delay'].rolling(window=15).min()

df3 = df1.merge(df2,left_index=True,right_index=True,how='outer')
# df3 = df1.join(df2,how='inner')

# df3 = df3.loc['2024-09-03 18:00:00':'2024-09-03 21:00:00']

plt.rcParams['figure.figsize'] = (20,6)

fig, ax = plt.subplots()

# 设置 x 轴的日期显示格式
# ax.xaxis.set_major_locator(mdates.HourLocator(interval=1))
ax.xaxis.set_major_locator(mdates.MinuteLocator(interval=30))
ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m-%d %H:%M'))

# 旋转 x 轴的日期标签，以免它们相互重叠
plt.xticks(rotation=75)

plt.plot(df3.index,df3['delay_x'], label='delay_x',color='b',alpha=0.8)
plt.plot(df3.index,df3['delay_y'], label='delay_zhuanxian',color='r',alpha=0.8)

# plt.xticks([])

plt.legend(loc='upper left',fontsize=12)
plt.grid(True)
plt.show()