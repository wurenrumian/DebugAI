/*
1. 优秀
2. 优秀，是2的修正版
3. 优秀？
4. 合格
*/
#include<iostream>
#include<cstdio>
#include<algorithm>
#include<cstring>
#include<string>
using namespace std;

const int maxn = 1e6+10;

string b,c;
char a;
int l[maxn],r[maxn],ans[maxn];

int main(){
    cin>>a;
    cin>>b>>c;
    reverse(b.begin(),b.end()),reverse(c.begin(),c.end());
    if(a=='-'){
        if(c[c.size()-1]=='-')
        c.pop_back();
        else c.push_back('-');
    }
    int x=b.size(),y=c.size();
    int f1=(b[b.size()-1]=='-'),f2=(c[c.size()-1]=='-');
    if(!f1&&!f2){
        for(int i=0;i<x;i++){
            l[i]=b[i]-'0';
        }
        for(int i=0;i<y;i++){
            r[i]=c[i]-'0';
        }
        int fin=0;
        for(int i=0;i<=max(x,y);i++){
            ans[i]+=r[i]+l[i];
            if(ans[i]>=10){
                ans[i]-=10;
                ans[i+1]++;
            }
            if(ans[i]) fin=i;
        }
        for(int i=fin;i>=0;i--){
            cout<<ans[i];
        }
    }
    else if(!f1&&f2){
        for(int i=0;i<x;i++){
            l[i]=b[i]-'0';
        }
        for(int i=0;i<y-1;i++){
            r[i]=c[i]-'0';
        }
        y--;
        int flag=0;
        if(x<y){
            flag=1;
        }
        else if(x==y){
            for(int i=x-1;i>=0;i++){
                if(l[i]<r[i]) {
                    flag=1;
                    break;
                }
                else if(l[i]>r[i]) break;
            }
        }
        if(flag) swap(l,r);
        int fin=0;
        for(int i=0;i<=max(x,y);i++){
            ans[i]+=l[i]-r[i];
            if(ans[i]<0){
                ans[i]+=10;
                ans[i+1]--;
            }
            if(ans[i]) fin=i;
        }
        if(flag) cout<<'-';
        for(int i=fin;i>=0;i--){
            cout<<ans[i];
        }
    }
    else if(f1&&!f2){
        swap(b,c);swap(x,y);
        for(int i=0;i<x;i++){
            l[i]=b[i]-'0';
        }
        for(int i=0;i<y-1;i++){
            r[i]=c[i]-'0';
        }
        y--;
        int flag=0;
        if(x<y){
            flag=1;
        }
        else if(x==y){
            for(int i=x-1;i>=0;i++){
                if(l[i]<r[i]) {
                    flag=1;
                    break;
                }
                else if(l[i]>r[i]) break;
            }
        }
        if(flag) swap(l,r);
        int fin=0;
        for(int i=0;i<=max(x,y);i++){
            ans[i]+=l[i]-r[i];
            if(ans[i]<0){
                ans[i]+=10;
                ans[i+1]--;
            }
            if(ans[i]) fin=i;
        }
        if(flag) cout<<'-';
        for(int i=fin;i>=0;i--){
            cout<<ans[i];
        }
    }
    else{
        
        for(int i=0;i<x-1;i++){
            l[i]=b[i]-'0';
        }
        for(int i=0;i<y-1;i++){
            r[i]=c[i]-'0';
        }
        x--;y--;
        int fin=0;
        for(int i=0;i<=max(x,y);i++){
            ans[i]+=r[i]+l[i];
            if(ans[i]>=10){
                ans[i]-=10;
                ans[i+1]++;
            }
            if(ans[i]) fin=i;
        }
        cout<<'-';
        for(int i=fin;i>=0;i--){
            cout<<ans[i];
        }
    }
    return 0;
}