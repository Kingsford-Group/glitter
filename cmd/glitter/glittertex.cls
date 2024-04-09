\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{r01preamble}[2018/10/19 R01 Preamble LaTeX class]
\LoadClass[11pt,oneside]{article}

% This documentclass implements the default formatting and typesetting
% environment for `glitter weave`

\RequirePackage[utf8]{inputenc}
\RequirePackage[T1]{fontenc}
\RequirePackage[letterpaper,centering,margin=1in]{geometry}
\RequirePackage{microtype}
\RequirePackage{secdot}
\RequirePackage{parskip}
\RequirePackage{xspace}
\RequirePackage{amssymb}
\RequirePackage{amsmath}
\RequirePackage{amsthm}
\RequirePackage{amsfonts}
\RequirePackage{booktabs}
\RequirePackage[dvipsnames]{xcolor}
\RequirePackage{mathptmx}
\RequirePackage[mode=match, group-separator={,}, tight-spacing=true]{siunitx}
\RequirePackage{url}
\RequirePackage{tikz}
\RequirePackage{listings}
\RequirePackage{hyperref}
\RequirePackage{tocloft}

\DeclareOption*{\PassOptionsToClass{\CurrentOption}{article}}
\ProcessOptions\relax

\urlstyle{rm}
\renewcommand{\ttdefault}{pcr}

\lstset{%
    language=Go,%
    basicstyle=\ttfamily\footnotesize,%
    texcl=true,%
    escapechar=\#,%
    extendedchars=true,%
    xleftmargin=2em,%
    showlines=false,
    numbers=left,
}

\widowpenalty=0
\clubpenalty=0

%% These commands are ignored by LaTeX, but read by glitter to defined
%% how various glitter components are typeset.

%%glitter Start         \documentclass{glittertex}
%%glitter StartBook     \glitterStartBook
%%glitter EndBook       \glitterEndBook
%%glitter StartText     \glitterStartText
%%glitter EndText       \glitterEndText$n
%%glitter StartCode     \glitterStartCode{$1}$n\begin{lstlisting}
%%glitter EndCode       \end{lstlisting}\glitterEndCode$n
%%glitter CodeEscape    #
%%glitter CodeCodeRef   #\glitterCodeRef{$1}#
%%glitter TextCodeRef   \glitterCodeRef{$1}
%%glitter CodeEscapeSub #\glitterHash#
%%glitter InlineCode    \lstinline@$1@
%%glitter AppendSymbol  \,+\kern-2pt

%%glitter CodeSet \glitterSet{append={$append},blocktable=${blocktable},labelbase={$labelbase},labelnum=$labelnum}

%%glitter WeaveLineRef  %%line $lineno "$filename"$n
%%glitter TangleLineRef /*line $filename:$lineno*/
%%glitter WeaveCommand  pdflatex ${WeaveFile}
%%glitter TangleCommand go build
\newlistof{block}{blk}{List of Key Blocks}

\global\def\glitterTrue{true}

\global\def\glitterAppendSymbol{}
\global\def\glitterBlockTable{false}
\global\def\glitterLabelBase{glitter:}
\global\def\glitterLabelNum{0}

\makeatletter
\define@key{glitterkeys}{append}[+]{\global\def\glitterAppendSymbol{#1}}
\define@key{glitterkeys}{blocktable}[true]{\global\def\glitterBlockTable{#1}}
\define@key{glitterkeys}{labelbase}{\global\def\glitterLabelBase{#1}}
\define@key{glitterkeys}{labelnum}{\global\def\glitterLabelNum{#1}}
\makeatother

\newcommand\glitterSet[1]{\setkeys{glitterkeys}{#1}}

\newcommand\glitterStartBook{\begin{document}}
\newcommand\glitterEndBook{\clearpage\listofblock\end{document}}
\newcommand\glitterStartText{\par}
\newcommand\glitterEndText{\par}
\newcommand\glitterCodeRef[1]{\ensuremath{\langle\kern-2pt\langle\textrm{#1}\rangle\kern-2pt\rangle}}
\newcommand\glitterStartCode[1]{\refstepcounter{block}%
    \par\noindent\hbox{$\glitterCodeRef{#1}\glitterAppendSymbol\equiv$}%
    \ifx\glitterTrue\glitterBlockTable\addcontentsline{blk}{block}{\protect\numberline{\theblock}#1}\fi}
\newcommand\glitterEndCode{\glitterSet{append={},blocktable=false,labelnum=0}}
\newcommand\glitterHash{\texttt{\char35}}
