#include<iostream>
using namespace std;
int a[11][11]={0},b[11][11]={0};
int n,m,g=0;
void myway(int i,int j)
{
    if (i==n&&j==m)
    {
        cout<<"YES";
        g=1;
        return;
    }
    if (a[i-1][j]+a[i+1][j]+a[i][j+1]+a[i][j-1]+b[i-1][j]+b[i+1][j]+b[i][j+1]+b[i][j-1]==0)
    {
        return;
    }
    if (a[i+1][j]==1&&g!=1)
    {
        b[i][j]=-1;
        myway(i+1,j);
        //b[i][j]=0;
    }
    if (a[i-1][j]==1&&g!=1)
    {
        b[i][j]=-1;
        myway(i-1,j);
        //b[i][j]=0;
    }
    if (a[i][j+1]==1&&g!=1)
    {
        b[i][j]=-1;
        myway(i,j+1);
        //b[i][j]=0;
    }
    if (a[i][j-1]==1&&g!=1)
    {
        b[i][j]=-1;
        myway(i,j-1);
        //b[i][j]=0;
    }
}
int main()
{

    cin>>n>>m;
    for (int i = 1; i <=n; i++)
    {
        for (int j = 1; j <= m; j++)
        {
            cin>>a[i][j];
        }
        
    }
    myway(1,1);
    if (g!=1)
    {
        cout<<"NO";
    }
    
    return 0;
}