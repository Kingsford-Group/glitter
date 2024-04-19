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
\RequirePackage[hyperindex=true,breaklinks=true,hidelinks,pdfa,pdfversion=2.0,plainpages=false]{hyperref}
\RequirePackage{tocloft}
\RequirePackage{graphicx}
\RequirePackage{booktabs}
\RequirePackage{enumerate}
\RequirePackage{enumitem}
\RequirePackage{siunitx}
\RequirePackage{makeidx}
\RequirePackage{natbib}
\RequirePackage{etoolbox}
\RequirePackage{xstring}

\DeclareOption*{\PassOptionsToClass{\CurrentOption}{article}}
\ProcessOptions\relax

\urlstyle{rm}
\renewcommand{\ttdefault}{pcr}

\usetikzlibrary{
    positioning, 
    matrix, 
    chains,
    calc,
    arrows.meta,
}

\lstset{%
    language=Go,%
    basicstyle=\ttfamily\footnotesize,%
    texcl=true,%
    escapechar=\@,%
    extendedchars=true,%
    xleftmargin=2em,%
    showlines=false,%
    showstringspaces=false,%
    numbers=left,%
    numberstyle=\tiny,
    tabsize=4,
    emptylines=2,
}

\widowpenalty=0
\clubpenalty=0

%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%
%%  Glitter Configuration
%%
%% These commands are ignored by LaTeX, but read by glitter to defined
%% how various glitter components are typeset. As written they are simply
%% the defaults.

% Events:
%%glitter Start         \documentclass{glittertex}
%%glitter StartBook     \glitterStartBook
%%glitter EndBook       \glitterEndBook
%%glitter StartText     \glitterStartText
%%glitter EndText       \glitterEndText$n
%%glitter StartCode     \glitterStartCode{$1}$n\begin{lstlisting}
%%glitter EndCode       \end{lstlisting}\glitterEndCode$n
%%glitter CodeEscape    @
%%glitter CodeCodeRef   @\glitterCodeRef{$blockid}{${name}}@
%%glitter TextCodeRef   \glitterCodeRef{$blockid}{${name}}
%%glitter EscapeSub     {\glitterHash}
%%glitter InlineCode    \lstinline\#$1#
%%glitter AppendSymbol  \,+\kern-2pt
%%glitter CodeSet       \glitterSet{append={$append},blocktable=$blocktable,blockid=$blockid,blockseries=$blockseries}
%%glitter WeaveLineRef  %%line "$filename":$lineno$n
%%glitter TangleLineRef /*line $filename:$lineno*/

%%not-used glitter Shell sh

% Settings:
%%glitter WeaveCommand  pdflatex "${weavefile}" && pdflatex "${weavefile}"
%%glitter TangleCommand go build

%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%
%%  Glitter-specific Macros
%%
%% These settings, variables, and macros are used by the default glitter
%% configuration. 

\newlistof{block}{blk}{List of Key Blocks}
\renewcommand\cftblktitlefont{\Large\bfseries}

% For \ifx to check for true value
\global\def\glitterTrue{true}

\global\def\glitterLabelBase{glitter-}

%%%%%%%%
%% Handling code block cross-references:
%%
\ExplSyntaxOn

% add an item to a list. creates the list if it does not exist.
\NewDocumentCommand{\listappend}{mm}
 {% #1 = list name, #2 = items
    \clist_if_exist:cTF { g_yannisl_list_#1_clist } 
    { } 
    {\clist_gclear_new:c { g_yannisl_list_#1_clist }}
    \clist_gput_right:cn { g_yannisl_list_#1_clist } { #2 }%
    \clist_gremove_duplicates:c { g_yannisl_list_#1_clist }%
 }

 % mark a reference. should be called anytime a code block is referenced.
\NewDocumentCommand{\xlabel}{m}%
{%
    \listappend{#1}{\thepage}%
    \makeatletter%
    \protected@write\@auxout{}{%
        \string\SiteRef{#1}{\clist_use:cn { g_yannisl_list_#1_clist} {,}}%
    }%
    \makeatother%
}

% print the cross references. Called from the code tha prints a code blcok.
\NewDocumentCommand{\xref}{m}%
{%
    \clist_if_exist:cTF { g_yannisl_listprev_#1_clist }%
    {%
        $\in$%
        \gdef\comma{}%
        \clist_map_inline:cn { g_yannis1_listprev_#1_clist } {\comma\pageref{#1}\gdef\comma{,}}%
    }%
    {}%
}

% set the complete cross references
\NewDocumentCommand{\SiteRef}{mm}{%
    \clist_gclear_new:c {g_yannisl_listprev_#1_clist}%
    \clist_gset:cn {g_yannisl_list_#1prev_clist} {#2}%
}

\ExplSyntaxOff

% Variables that affect typesetting certain blocks.
\global\def\glitterAppendSymbol{}
\global\def\glitterBlockTable{false}
\global\def\glitterBlockId{0}
\global\def\glitterBlockSeries{1}

% "Functions" called by the keyval package from \glitterSet. 
\makeatletter
\define@key{glitterkeys}{append}{\global\def\glitterAppendSymbol{#1}}
\define@key{glitterkeys}{blocktable}{\global\def\glitterBlockTable{#1}}
\define@key{glitterkeys}{blockid}{\global\def\glitterBlockId{#1}}
\define@key{glitterkeys}{blockseries}{\global\def\glitterBlockSeries{#1}}
\newcommand\iflabelexists[2]{\@ifundefined{r@#1}{}{#2}}
\makeatother

% Emitted by Weave at "CodeSet" events. Used to determine how the next
% code block looks.
\newcommand\glitterSet[1]{\setkeys{glitterkeys}{#1}}

\def\ifempty#1{\def\temp{#1}\ifx\temp\empty}

% Embitted by Weave at various events.
\newcommand\glitterStartBook{\begin{document}}
\newcommand\glitterEndBook{\clearpage\listofblock\clearpage\phantomsection\printindex\end{document}}
\newcommand\glitterStartText{\par}
\newcommand\glitterEndText{\par}

\newcommand\glitterCodeRef[2]{%
    \ifempty{#1}\def\pp{}\else\def\pp{\ \pageref{\glitterLabelBase#1-0}\xlabel{\glitterLabelBase#1-0}}\fi%
    \ensuremath{%
    \langle\kern-2pt\langle%
    \textrm{#2\tiny\pp}%
    \rangle\kern-2pt\rangle}}

\newcommand\glitterStartCode[1]{\refstepcounter{block}\label{\glitterLabelBase\glitterBlockId-\glitterBlockSeries}% 
    \par\noindent\hbox{$\glitterCodeRef{}{#1}\glitterAppendSymbol\equiv$}\hfill%
    {\small\xref{\glitterLabelBase\glitterBlockId-\glitterBlockSeries}}%
    \ifnum\glitterBlockSeries>0{%
        \def\glitterPrev{\the\numexpr\glitterBlockSeries - 1}%
        $\vartriangle${\small\pageref{\glitterLabelBase\glitterBlockId-\glitterPrev}}}\fi%
    \def\glitterNext{\the\numexpr\glitterBlockSeries + 1}%
    \iflabelexists{\glitterLabelBase\glitterBlockId-\glitterNext}{%
        $\triangledown${\small\pageref{\glitterLabelBase\glitterBlockId-\glitterNext}}%
    }%
    \ifx\glitterTrue\glitterBlockTable\addcontentsline{blk}{block}{\protect#1}\fi}

\newcommand\glitterEndCode{\glitterSet{append={},blocktable=false,blockid=0,blockseries=1}}

\newcommand\glitterHash{\texttt{\char64}}

\makeindex

% end
