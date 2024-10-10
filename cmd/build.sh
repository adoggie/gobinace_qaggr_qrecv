
AGG_VER=0.3.6
RECV_VER=0.5.1

INSTALL_DIR=`pwd`/../bin/ccsuits_${AGG_VER}_${RECV_VER}
export {http,https,ftp}_proxy=http://wallizard.com:59128

#rm -rf ../bin

mkdir -p ${INSTALL_DIR}/qaggr
mkdir -p ${INSTALL_DIR}/qrecv

cd Forward
go build -o ${INSTALL_DIR}/forward-go

cd ..

cd QuoteAggrService
go build -o ${INSTALL_DIR}/qaggr/qaggr_bn_${AGG_VER}
cp *.json ${INSTALL_DIR}/qaggr
cp *.toml ${INSTALL_DIR}/qaggr

cd ..

echo `pwd`
cd QuoteRecvSerivce
go build -o ${INSTALL_DIR}/qrecv/qrecv_bn_${RECV_VER}
cp *.toml ${INSTALL_DIR}/qrecv

cd ..
tar cvzf ccsuits_${AGG_VER}_${RECV_VER}.tar.gz ${INSTALL_DIR}
scp -P 19015 ccsuits_${AGG_VER}_${RECV_VER}.tar.gz bzz@wallizard.com:/home/bzz/website/elabs/download/

# jp - dev
#sshpass -p "${PASS}" rsync -avr -e "ssh -p 22 -o StrictHostKeyChecking=no" ${INSTALL_DIR} scott@193.32.149.156:/home/scott/bin/
## jp - prod1
#sshpass -p "${PASS}" rsync -avr -e "ssh -p 22 -o StrictHostKeyChecking=no" ${INSTALL_DIR} scott@cc-jp1:/home/scott/bin/
##jp -prod2
#sshpass -p "${PASS}" rsync -avr -e "ssh -p 22 -o StrictHostKeyChecking=no" ${INSTALL_DIR} scott@cc-jp2:/home/scott/bin/
##sshpass -p "${PASS}" rsync -avr -e "ssh -p 30022 -o StrictHostKeyChecking=no" ${INSTALL_DIR} scott@203.156.254.197:/home/scott/bin/
##jp - subaohui
#sshpass -p "${PASS}" rsync -avr -e "ssh -p 22 -o StrictHostKeyChecking=no" ${INSTALL_DIR} scott@193.32.151.225:/home/scott/bin/
#
#sshpass -p "${PASS}" rsync -avr -e "ssh -p 30022 -o StrictHostKeyChecking=no" ${INSTALL_DIR} root@203.156.254.197:/root/bin/
#sshpass -p "${PASS}" rsync -avr -e "ssh -p 40022 -o StrictHostKeyChecking=no" ${INSTALL_DIR} root@203.156.254.197:/root/bin/