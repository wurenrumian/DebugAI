/*
10^100是101位，加上/0，/n
1. 优秀
2. 合格，对字符串掌握不好，strlen应用有问题
3. 优秀
4. 合格
*/
#include <stdio.h>
#include <string.h>
int main()
{
    char x[100],y[100];
    int m,n,i;
    gets(x);
    scanf("\n");
    gets(y);
    m=strlen(x);
    n=strlen(y);
    for(i=0;i<=m;i++)
    {
        printf("%c",x[i]);
    }
    printf("\n");
    for(i=0;i<=n;i++)
    {
        printf("%c",y[i]);
    }
    printf("\n");
    return 0;
}