# ENEL query flow
##### _Read it in english [here](https://github.com/ozzono/enel_invoice/blob/master/README.md)._
Esse pacote usa o [chromedp](github.com/chromedp/chromedp) para navegar pelo página de usuário da [enel](https://portalhome.eneldistribuicaosp.com.br/#/login)
Esse pacote retorna os seguintes valores:
- vencimento _(dueDate)_
- status _(status)_
- código de barras_(barCode)_
- valor _(value)_