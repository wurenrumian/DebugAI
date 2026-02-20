/*
1. 合格，10未对应为X
2. 优秀
3. 优秀
4. 优秀
*/
#include<bits/stdc++.h>
using namespace std;
const int N=22, rem=11;
char a[N];
int main()
{
    cin >> a;
    int len=strlen(a);
    int ans=0; int num=0;
    for(int i=0; i<len-1; ++i)
    {
        if(a[i] != '-')
        {
            ++num;
            ans = (ans + num*(a[i]-'0')) % rem;
        }
    }
    if(a[len-1] == ans+'0')
    {
        cout << "Right";
    }
    else
    {
        for(int i=0; i<len-1; ++i) 
            cout << a[i];
        cout << ans;
    }
    return 0;
}