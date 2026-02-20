/*
1. 合格，没以16进制输出
2. 合格，有一处漏洞
3. 优秀
4. 优秀
*/
#include <iostream>
using namespace std;
int main()
{
    int n;
    int mix=0;
    int count;
    int a[6];
    cin >> n;
    for (int i = 0; i < n; i++)
    {
        bool live=1;
        for (int j = 0; j < 6; j++)
        {
            cin >> a[j];
            int num1=0,num2=0,num3=0,num4=0,num5=0,num6=0;
        for(int l=0;l<6;l++)
        {
            if(a[l]==2)//错了
            num1++;
            if(a[l]==2)
            num2++;
            if(a[l]==3)
            num3++;
            if (a[l]==4)
            num4++;
            if(a[l]==5)
            num5++;
            if(a[l]==6)
            num6++;
        }
        if(num4==4 && num1==2)
        mix+=2048;
        else if(num4==6)
        mix+=1024;
        else if(num1==6)
        mix+=512;
        else if(num2==6)
        mix+=256;
        else if(num4==5)
        mix+=128;
        else if(num2==5)
        mix+=64;
        else if(num4==4)
        mix+=32;
        else if(num1==1&&num2==1&&num3==1&&num4==1&&num5==1&&num6==1)
        mix+=16;
        else if(num4==3)
        mix+=8;
        else if(num2==4)
        mix+=4;
        else if(num4==2)
        mix+=2;
        else if(num4==1)
        mix+=1;
        else
        bool live=0;
        }
        if(live==0)
        break;
    }
    cout<<mix<<endl;//输出格式
}