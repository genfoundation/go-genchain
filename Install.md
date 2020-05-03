## gen链挖矿使用说明

### Windows下的安装使用

**1. 下载ggen.exe文件** [点击下载](http://www.xxx.com/ggen.exe)
**2. 在cmd命令窗口运行ggen.exe**
        ```ggen console ```
**3. 设置挖矿账号，第一次运行需创建一个账号用来接受挖矿时奖励的gen币**
   * 在ggen控制台命令窗口创建地址
   ```
   > personal.newAccount('123123') //123123为账号密码，请自行修改。
   ```
   * 使用gen钱包创建账号地址，如这样的地址`0xa0d9b74ecb7c48e29162a5503e044135d64f26b3`，创建好地址后，在ggen控制台窗口执行下列命令。
   ```
   > miner.setEtherbase('0xa0d9b74ecb7c48e29162a5503e044135d64f26b3')
   ```
**4. 开始挖矿**
   ```
   > miner.start()
   ```